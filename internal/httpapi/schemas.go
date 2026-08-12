// schemas.go — Go equivalents of the marshmallow schemas in blueprints/.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §6.2.
//
// Every validation message emitted here was captured from the running Flask app
// (see testdata/golden/e1*_*.json) rather than transcribed from the PRD.
// Optional fields are pointers so that marshmallow's load_default=None round
// trips as JSON null instead of a zero value.

package httpapi

// ── /auth/signup, /auth/login ───────────────────────────────────────────────

type SignupRequest struct {
	MobileNumber *string `json:"mobile_number" validate:"required,len=10,numeric"`
	Password     *string `json:"password"      validate:"required,mlen=6:"`
	Name         *string `json:"name"`
}

func (SignupRequest) Message(field, tag string) string {
	switch field {
	case "mobile_number":
		if tag == "len" || tag == "numeric" {
			return "Mobile number must be exactly 10 digits."
		}
	case "password":
		if tag == "mlen" {
			return "Password must be at least 6 characters."
		}
	}
	return ""
}

type LoginRequest struct {
	MobileNumber *string `json:"mobile_number" validate:"required,len=10,numeric"`
	Password     *string `json:"password"      validate:"required,mlen=1:"`
}

func (LoginRequest) Message(field, tag string) string {
	switch field {
	case "mobile_number":
		if tag == "len" || tag == "numeric" {
			return "Mobile number must be exactly 10 digits."
		}
	case "password":
		if tag == "mlen" {
			return "Password is required."
		}
	}
	return ""
}

// ── /farmer/basic-details, /farmer/location ─────────────────────────────────

type BasicDetailsRequest struct {
	Name              *string `json:"name"               validate:"required,mlen=1:255"`
	Age               *int    `json:"age"                validate:"omitempty,mrange=1:120"`
	Gender            *string `json:"gender"             validate:"omitempty,oneof=male female other"`
	PreferredLanguage *string `json:"preferred_language" validate:"required,oneof=english hindi marathi others"`
}

type LocationRequest struct {
	PinCode     *string `json:"pin_code"     validate:"required,len=6,numeric"`
	VillageName *string `json:"village_name" validate:"required,mlen=1:255"`
	FullAddress *string `json:"full_address"`
}

func (LocationRequest) Message(field, tag string) string {
	if field == "pin_code" && (tag == "len" || tag == "numeric") {
		return "pin_code must be exactly 6 digits."
	}
	return ""
}

// ── /farm ───────────────────────────────────────────────────────────────────

type FarmRequest struct {
	FarmName         *string  `json:"farm_name"      validate:"required,mlen=1:255"`
	TotalArea        *float64 `json:"total_area"     validate:"required,mrange=0.01:"`
	AreaUnit         *string  `json:"area_unit"      validate:"required,oneof=acres hectares"`
	LandOwnership    *string  `json:"land_ownership" validate:"required,oneof=own_land leased_land contract_farming"`
	Latitude         *float64 `json:"latitude"       validate:"required,mrange=-90:90"`
	Longitude        *float64 `json:"longitude"      validate:"required,mrange=-180:180"`
	BoundaryGeom     any      `json:"boundary_geom"`
	LocationPhotoURL *string  `json:"location_photo_url"`
}

// ── /crop, /irrigation, /soil, /consent ─────────────────────────────────────

type CropRequest struct {
	FarmID               *string `json:"farm_id"      validate:"required,uuid"`
	CropName             *string `json:"crop_name"    validate:"required,mlen=1:255"`
	CropVariety          *string `json:"crop_variety"`
	SowingDate           *string `json:"sowing_date"  validate:"required,datetime=2006-01-02"`
	Season               *string `json:"season"       validate:"required,oneof=kharif rabi zaid"`
	ExpectedHarvestMonth *string `json:"expected_harvest_month"`
}

type IrrigationRequest struct {
	FarmID         *string `json:"farm_id"         validate:"required,uuid"`
	IrrigationType *string `json:"irrigation_type" validate:"required,oneof=rainfed borewell canal drip_irrigation sprinkler"`
	WaterSource    *string `json:"water_source"`
}

type SoilRequest struct {
	FarmID   *string `json:"farm_id"   validate:"required,uuid"`
	SoilType *string `json:"soil_type" validate:"required,oneof=black red sandy mixed unknown"`
}

type ConsentRequest struct {
	// A pointer so that an absent key is "Missing data for required field."
	// rather than silently defaulting to false.
	SatelliteMonitoring *bool `json:"satellite_monitoring" validate:"required"`
}

// ── analysis endpoints ──────────────────────────────────────────────────────
//
// There is deliberately no schema type here. The analysis routes are
// hand-validated: app.py checks for the presence of "geometry" itself and then
// calls validate_polygon, producing the {"error": …} envelope rather than the
// marshmallow one. See requireGeometry in analyze_handler.go.
