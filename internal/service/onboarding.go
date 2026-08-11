// Package service holds the onboarding business logic, auth, SMS and the PIN
// code client, mirroring backend/services/.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §7.10, §8.
//
// Every onboarding function does the same three things in the same order:
// (a) an optional ownership check, (b) the SQL write, (c) a BEST-EFFORT
// Firestore session write. Firestore failures are logged at WARN and never fail
// the request.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/crypto"
	"github.com/SanTiwari07/NDVI_satellite/internal/firestore"
	"github.com/SanTiwari07/NDVI_satellite/internal/jwtutil"
	"github.com/SanTiwari07/NDVI_satellite/internal/repo"
)

// ErrOwnership is returned when a farm does not belong to the caller. The
// handler maps it to 403 with the message §7.10 specifies.
var ErrOwnership = errors.New("farm ownership")

// Auth-flow sentinel errors. Each maps to a specific status and message in
// §10.9, so they must stay distinguishable.
var (
	ErrMobileTaken   = errors.New("Mobile number already registered. Please log in.")
	ErrNoAccount     = errors.New("No account found with this mobile number. Please sign up.")
	ErrNoPassword    = errors.New("Account has no password set. Please sign up again.")
	ErrWrongPassword = errors.New("Incorrect password. Please try again.")
)

// Onboarding bundles the dependencies the nine steps need.
type Onboarding struct {
	Cfg    *config.Config
	Store  *repo.Store
	FS     *firestore.Client
	JWT    *jwtutil.Issuer
	PinAPI *PinCodeClient
	Log    *slog.Logger
}

// ── Step 1: signup / login ──────────────────────────────────────────────────

// AuthResult is the body both /auth/signup and /auth/login return.
type AuthResult struct {
	Token     string `json:"token"`
	IsNewUser bool   `json:"is_new_user"`
	FarmerID  string `json:"farmer_id"`
}

// Signup creates a farmer and issues a token.
func (o *Onboarding) Signup(ctx context.Context, mobile, password string, name *string) (*AuthResult, error) {
	existing, err := o.Store.FindFarmerByMobile(ctx, mobile)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrMobileTaken
	}

	// Werkzeug-format scrypt, so the row stays verifiable by the Python
	// backend if this deployment is rolled back (§8.2).
	hash, err := crypto.GeneratePassword(password)
	if err != nil {
		return nil, err
	}

	farmer, err := o.Store.CreateFarmer(ctx, mobile, hash, name)
	if err != nil {
		return nil, err
	}

	token, err := o.JWT.Issue(farmer.ID)
	if err != nil {
		return nil, err
	}

	o.FS.WriteSession(ctx, farmer.ID, 1, map[string]any{"mobile_number": mobile})
	o.Log.Info("[Auth] Signup complete for farmer: " + farmer.ID)

	return &AuthResult{Token: token, IsNewUser: true, FarmerID: farmer.ID}, nil
}

// Login verifies the password and issues a token.
//
// The three failure modes are distinct messages in §10.9 and must not be
// collapsed into one "invalid credentials", however much better that would be
// for security — the frontend branches on them.
func (o *Onboarding) Login(ctx context.Context, mobile, password string) (*AuthResult, error) {
	farmer, err := o.Store.FindFarmerByMobile(ctx, mobile)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrNoAccount
	}
	if err != nil {
		return nil, err
	}
	if farmer.PasswordHash == nil || *farmer.PasswordHash == "" {
		return nil, ErrNoPassword
	}

	ok, err := crypto.VerifyPassword(*farmer.PasswordHash, password)
	if err != nil {
		// A corrupt hash is a server-side problem, not a wrong password.
		return nil, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return nil, ErrWrongPassword
	}

	token, err := o.JWT.Issue(farmer.ID)
	if err != nil {
		return nil, err
	}

	// is_new_user means "onboarding not finished", inferred from a blank name.
	isNew := farmer.Name == nil || *farmer.Name == ""
	o.Log.Info("[Auth] Login successful for farmer: " + farmer.ID)

	return &AuthResult{Token: token, IsNewUser: isNew, FarmerID: farmer.ID}, nil
}

// ── Step 2: basic details ───────────────────────────────────────────────────

// UpdateBasicDetails saves the farmer profile.
func (o *Onboarding) UpdateBasicDetails(
	ctx context.Context, farmerID, name string, age *int, gender *string, language string,
) (*repo.BasicDetails, error) {
	updated, err := o.Store.UpdateFarmerDetails(ctx, farmerID, name, age, gender, language)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, fmt.Errorf("Farmer %s not found.", farmerID)
	}
	if err != nil {
		return nil, err
	}
	o.FS.WriteSession(ctx, farmerID, 2, map[string]any{
		"name": name, "preferred_language": language,
	})
	o.Log.Info("[Farmer] Basic details saved for farmer " + farmerID)
	return updated, nil
}

// ── Step 3: location ────────────────────────────────────────────────────────

// SaveLocation resolves the PIN then writes the location row.
//
// §7.10: on ANY PIN-lookup failure the geo fields become empty strings and the
// request still succeeds. The lookup must never fail the write.
func (o *Onboarding) SaveLocation(
	ctx context.Context, farmerID, pinCode, village string, fullAddress *string,
) (*repo.LocationRow, error) {
	var state, district, taluka string
	if geo, err := o.PinAPI.Resolve(ctx, pinCode); err != nil {
		o.Log.Warn(fmt.Sprintf(
			"[Farmer] PIN resolve failed (%v) — saving with empty geo fields", err))
	} else {
		state, district, taluka = geo.State, geo.District, geo.Taluka
	}

	location, err := o.Store.CreateFarmerLocation(
		ctx, farmerID, pinCode, state, district, taluka, village, fullAddress)
	if err != nil {
		return nil, err
	}

	o.FS.WriteSession(ctx, farmerID, 3, map[string]any{
		"pin_code": pinCode, "village_name": village,
		"state": state, "district": district,
	})
	o.Log.Info(fmt.Sprintf("[Farmer] Location saved for farmer %s (PIN=%s)", farmerID, pinCode))
	return location, nil
}

// ── Steps 4+5: farm ─────────────────────────────────────────────────────────

// CreateFarm saves the farm and its optional PostGIS boundary.
func (o *Onboarding) CreateFarm(
	ctx context.Context, farmerID, farmName string, totalArea float64,
	areaUnit, landOwnership string, latitude, longitude float64,
	boundary json.RawMessage, photoURL *string,
) (*repo.FarmRow, error) {
	farm, err := o.Store.CreateFarm(ctx, farmerID, farmName, totalArea, areaUnit,
		landOwnership, latitude, longitude, boundary, photoURL)
	if err != nil {
		return nil, err
	}
	o.FS.WriteSession(ctx, farmerID, 5, map[string]any{
		"farm_id": farm.ID, "farm_name": farmName,
		"latitude": latitude, "longitude": longitude,
	})
	o.Log.Info(fmt.Sprintf("[Farm] Farm '%s' created for farmer %s (farm_id=%s)",
		farmName, farmerID, farm.ID))
	return farm, nil
}

// assertOwnership is the shared check for steps 6-8.
//
// A missing farm and a farm belonging to someone else produce the SAME message,
// which avoids leaking whether a given farm id exists.
func (o *Onboarding) assertOwnership(ctx context.Context, farmerID, farmID string) error {
	farm, err := o.Store.GetFarmByID(ctx, farmID)
	if errors.Is(err, repo.ErrNotFound) || (err == nil && farm.FarmerID != farmerID) {
		return fmt.Errorf("%w: Farm %s not found or does not belong to this farmer.",
			ErrOwnership, farmID)
	}
	return err
}

// OwnershipMessage extracts the user-facing text from an ownership error.
func OwnershipMessage(farmID string) string {
	return fmt.Sprintf("Farm %s not found or does not belong to this farmer.", farmID)
}

// ── Step 6: crop ────────────────────────────────────────────────────────────

// AddCrop validates ownership then inserts the crop.
func (o *Onboarding) AddCrop(
	ctx context.Context, farmerID, farmID, cropName string, variety *string,
	sowingDate, season string, harvestMonth *string,
) (*repo.CropRow, error) {
	if err := o.assertOwnership(ctx, farmerID, farmID); err != nil {
		return nil, err
	}
	crop, err := o.Store.CreateCrop(ctx, farmID, cropName, variety, sowingDate, season, harvestMonth)
	if err != nil {
		return nil, err
	}
	o.FS.WriteSession(ctx, farmerID, 6, map[string]any{"crop_name": cropName})
	o.Log.Info(fmt.Sprintf("[Crop] '%s' added to farm %s", cropName, farmID))
	return crop, nil
}

// ── Step 7: irrigation ──────────────────────────────────────────────────────

// AddIrrigation validates ownership then inserts the irrigation record.
func (o *Onboarding) AddIrrigation(
	ctx context.Context, farmerID, farmID, irrigationType string, waterSource *string,
) (*repo.IrrigationRow, error) {
	if err := o.assertOwnership(ctx, farmerID, farmID); err != nil {
		return nil, err
	}
	row, err := o.Store.CreateIrrigation(ctx, farmID, irrigationType, waterSource)
	if err != nil {
		return nil, err
	}
	o.FS.WriteSession(ctx, farmerID, 7, map[string]any{"irrigation_type": irrigationType})
	o.Log.Info(fmt.Sprintf("[Irrigation] %s added to farm %s", irrigationType, farmID))
	return row, nil
}

// ── Step 8: soil (optional) ─────────────────────────────────────────────────

// AddSoilInfo validates ownership then upserts the soil record.
func (o *Onboarding) AddSoilInfo(
	ctx context.Context, farmerID, farmID, soilType string,
) (*repo.SoilRow, error) {
	if err := o.assertOwnership(ctx, farmerID, farmID); err != nil {
		return nil, err
	}
	row, err := o.Store.UpsertSoilInfo(ctx, farmID, soilType)
	if err != nil {
		return nil, err
	}
	o.FS.WriteSession(ctx, farmerID, 8, map[string]any{"soil_type": soilType})
	o.Log.Info(fmt.Sprintf("[Soil] %s soil type recorded for farm %s", soilType, farmID))
	return row, nil
}

// ── Step 9: consent ─────────────────────────────────────────────────────────

// ConsentResult is the body POST /consent returns.
type ConsentResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ConsentID string `json:"consent_id"`
}

// SubmitConsent finishes onboarding.
//
// Three effects, in order: upsert consent, delete the Firestore session, then
// for every farm log the VI Engine trigger and write a placeholder alert.
// §7.10 says the trigger is a mock today and must STAY a mock — writing real
// values here would mean running the pipeline synchronously inside the consent
// request.
func (o *Onboarding) SubmitConsent(ctx context.Context, farmerID string, satellite bool) (*ConsentResult, error) {
	consent, err := o.Store.CreateConsent(ctx, farmerID, satellite)
	if err != nil {
		return nil, err
	}

	o.FS.DeleteSession(ctx, farmerID)

	farms, err := o.Store.GetFarmsByFarmer(ctx, farmerID)
	if err != nil {
		return nil, err
	}
	for _, farm := range farms {
		o.Log.Info("[VI Engine] VI Engine triggered for farm_id: " + farm.ID)
		o.FS.WriteFarmAlert(ctx, farm.ID, firestore.VIData{CVIMean: 0.0, NDVI: 0.0, NDMI: 0.0})
	}

	o.Log.Info("[Consent] Onboarding complete for farmer " + farmerID)
	return &ConsentResult{
		Success:   true,
		Message:   "Onboarding complete",
		ConsentID: consent.ID,
	}, nil
}
