package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Farmer is a row of the farmers table.
//
// PasswordHash is a pointer because Firebase-created rows have it NULL, which
// login must distinguish from "wrong password" (§10.9).
type Farmer struct {
	ID                string
	MobileNumber      string
	PasswordHash      *string
	Name              *string
	Age               *int
	Gender            *string
	PreferredLanguage *string
	CreatedAt         pgtype.Timestamptz
}

// FarmerLocation is a row of farmer_locations.
type FarmerLocation struct {
	ID          string
	FarmerID    string
	PinCode     string
	State       string
	District    string
	Taluka      string
	VillageName string
}

// FindFarmerByMobile returns the farmer including the password hash.
func (s *Store) FindFarmerByMobile(ctx context.Context, mobile string) (*Farmer, error) {
	var f Farmer
	err := s.pool.QueryRow(ctx,
		`SELECT id, mobile_number, password_hash, name, preferred_language
		 FROM farmers WHERE mobile_number = $1`, mobile,
	).Scan(&f.ID, &f.MobileNumber, &f.PasswordHash, &f.Name, &f.PreferredLanguage)
	if err != nil {
		return nil, wrap("find farmer by mobile", err)
	}
	return &f, nil
}

// FindFarmerByID returns the farmer profile used by /dashboard.
func (s *Store) FindFarmerByID(ctx context.Context, farmerID string) (*Farmer, error) {
	var f Farmer
	err := s.pool.QueryRow(ctx,
		`SELECT id, mobile_number, name, age, gender, preferred_language
		 FROM farmers WHERE id = $1`, farmerID,
	).Scan(&f.ID, &f.MobileNumber, &f.Name, &f.Age, &f.Gender, &f.PreferredLanguage)
	if err != nil {
		return nil, wrap("find farmer by id", err)
	}
	return &f, nil
}

// CreateFarmer inserts a new farmer. name is optional at signup.
func (s *Store) CreateFarmer(ctx context.Context, mobile, passwordHash string, name *string) (*Farmer, error) {
	var f Farmer
	err := s.pool.QueryRow(ctx,
		`INSERT INTO farmers (mobile_number, password_hash, name)
		 VALUES ($1, $2, $3)
		 RETURNING id, mobile_number, name, preferred_language, created_at`,
		mobile, passwordHash, name,
	).Scan(&f.ID, &f.MobileNumber, &f.Name, &f.PreferredLanguage, &f.CreatedAt)
	if err != nil {
		return nil, wrap("create farmer", err)
	}
	return &f, nil
}

// BasicDetails is the UPDATE … RETURNING row /farmer/basic-details echoes.
type BasicDetails struct {
	ID                string  `json:"id"`
	Name              *string `json:"name"`
	Age               *int    `json:"age"`
	Gender            *string `json:"gender"`
	PreferredLanguage *string `json:"preferred_language"`
}

// UpdateFarmerDetails saves step 2 of onboarding.
func (s *Store) UpdateFarmerDetails(
	ctx context.Context, farmerID, name string, age *int, gender *string, language string,
) (*BasicDetails, error) {
	var d BasicDetails
	err := s.pool.QueryRow(ctx,
		`UPDATE farmers
		 SET name = $1, age = $2, gender = $3, preferred_language = $4, updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, name, age, gender, preferred_language`,
		name, age, gender, language, farmerID,
	).Scan(&d.ID, &d.Name, &d.Age, &d.Gender, &d.PreferredLanguage)
	if err != nil {
		return nil, wrap("update farmer details", err)
	}
	return &d, nil
}

// LocationRow is the INSERT … RETURNING row /farmer/location echoes.
type LocationRow struct {
	ID          string `json:"id"`
	FarmerID    string `json:"farmer_id"`
	PinCode     string `json:"pin_code"`
	State       string `json:"state"`
	District    string `json:"district"`
	Taluka      string `json:"taluka"`
	VillageName string `json:"village_name"`
}

// CreateFarmerLocation saves step 3 of onboarding.
func (s *Store) CreateFarmerLocation(
	ctx context.Context, farmerID, pinCode, state, district, taluka, village string, fullAddress *string,
) (*LocationRow, error) {
	var l LocationRow
	err := s.pool.QueryRow(ctx,
		`INSERT INTO farmer_locations
		   (farmer_id, pin_code, state, district, taluka, village_name, full_address)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, farmer_id, pin_code, state, district, taluka, village_name`,
		farmerID, pinCode, state, district, taluka, village, fullAddress,
	).Scan(&l.ID, &l.FarmerID, &l.PinCode, &l.State, &l.District, &l.Taluka, &l.VillageName)
	if err != nil {
		return nil, wrap("create farmer location", err)
	}
	return &l, nil
}
