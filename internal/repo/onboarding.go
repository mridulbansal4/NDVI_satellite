package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── crops ───────────────────────────────────────────────────────────────────

// CropRow is the INSERT … RETURNING row POST /crop echoes.
type CropRow struct {
	ID                   string  `json:"id"`
	FarmID               string  `json:"farm_id"`
	CropName             string  `json:"crop_name"`
	CropVariety          *string `json:"crop_variety"`
	SowingDate           string  `json:"sowing_date"`
	Season               string  `json:"season"`
	ExpectedHarvestMonth *string `json:"expected_harvest_month"`
	CreatedAt            string  `json:"created_at"`
}

// Crop is the dashboard projection of a crop row.
type Crop struct {
	CropName   string `json:"crop_name"`
	Season     string `json:"season"`
	SowingDate string `json:"sowing_date"`
}

// CreateCrop inserts a crop for a farm (step 6).
func (s *Store) CreateCrop(
	ctx context.Context, farmID, cropName string, variety *string,
	sowingDate, season string, harvestMonth *string,
) (*CropRow, error) {
	var (
		c         CropRow
		sowing    pgtype.Date
		createdAt pgtype.Timestamptz
	)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO crops
		   (farm_id, crop_name, crop_variety, sowing_date, season, expected_harvest_month)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, farm_id, crop_name, crop_variety, sowing_date,
		           season, expected_harvest_month, created_at`,
		farmID, cropName, variety, sowingDate, season, harvestMonth,
	).Scan(&c.ID, &c.FarmID, &c.CropName, &c.CropVariety, &sowing,
		&c.Season, &c.ExpectedHarvestMonth, &createdAt)
	if err != nil {
		return nil, wrap("create crop", err)
	}
	// RFC 1123, not YYYY-MM-DD: the POST response returns the raw date object
	// and Flask renders it that way (golden e17_crop_success.json). The
	// dashboard projection below uses dateString instead.
	c.SowingDate = rfc1123Date(sowing)
	if createdAt.Valid {
		c.CreatedAt = createdAt.Time.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	}
	return &c, nil
}

// GetCropsByFarm returns a farm's crops, newest sowing first.
//
// The dashboard calls this once per farm — an N+1 that is preserved for parity
// and filed as K7.
func (s *Store) GetCropsByFarm(ctx context.Context, farmID string) ([]Crop, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT crop_name, season, sowing_date
		 FROM crops WHERE farm_id = $1 ORDER BY sowing_date DESC`, farmID)
	if err != nil {
		return nil, wrap("get crops by farm", err)
	}
	defer rows.Close()

	out := []Crop{}
	for rows.Next() {
		var (
			c      Crop
			sowing pgtype.Date
		)
		if err := rows.Scan(&c.CropName, &c.Season, &sowing); err != nil {
			return nil, wrap("scan crop", err)
		}
		c.SowingDate = dateString(sowing)
		out = append(out, c)
	}
	return out, wrap("iterate crops", rows.Err())
}

// ── irrigation ──────────────────────────────────────────────────────────────

// IrrigationRow is the INSERT … RETURNING row POST /irrigation echoes.
type IrrigationRow struct {
	ID             string  `json:"id"`
	FarmID         string  `json:"farm_id"`
	IrrigationType string  `json:"irrigation_type"`
	WaterSource    *string `json:"water_source"`
}

// CreateIrrigation inserts an irrigation record (step 7).
func (s *Store) CreateIrrigation(
	ctx context.Context, farmID, irrigationType string, waterSource *string,
) (*IrrigationRow, error) {
	var r IrrigationRow
	err := s.pool.QueryRow(ctx,
		`INSERT INTO irrigation (farm_id, irrigation_type, water_source)
		 VALUES ($1, $2, $3)
		 RETURNING id, farm_id, irrigation_type, water_source`,
		farmID, irrigationType, waterSource,
	).Scan(&r.ID, &r.FarmID, &r.IrrigationType, &r.WaterSource)
	if err != nil {
		return nil, wrap("create irrigation", err)
	}
	return &r, nil
}

// ── soil ────────────────────────────────────────────────────────────────────

// SoilRow is the INSERT … RETURNING row POST /soil echoes.
type SoilRow struct {
	ID       string `json:"id"`
	FarmID   string `json:"farm_id"`
	SoilType string `json:"soil_type"`
}

// UpsertSoilInfo inserts or updates soil info (step 8), keyed on the UNIQUE
// farm_id constraint so a farmer can revise their answer.
func (s *Store) UpsertSoilInfo(ctx context.Context, farmID, soilType string) (*SoilRow, error) {
	var r SoilRow
	err := s.pool.QueryRow(ctx,
		`INSERT INTO soil_info (farm_id, soil_type)
		 VALUES ($1, $2)
		 ON CONFLICT (farm_id) DO UPDATE SET soil_type = EXCLUDED.soil_type
		 RETURNING id, farm_id, soil_type`,
		farmID, soilType,
	).Scan(&r.ID, &r.FarmID, &r.SoilType)
	if err != nil {
		return nil, wrap("upsert soil info", err)
	}
	return &r, nil
}

// ── consents ────────────────────────────────────────────────────────────────

// ConsentRow is the INSERT … RETURNING row behind POST /consent.
type ConsentRow struct {
	ID                  string
	FarmerID            string
	SatelliteMonitoring bool
}

// CreateConsent upserts the consent record (step 9), so a resubmission is
// idempotent rather than a unique-violation.
func (s *Store) CreateConsent(ctx context.Context, farmerID string, satellite bool) (*ConsentRow, error) {
	var r ConsentRow
	var consentedAt pgtype.Timestamptz
	err := s.pool.QueryRow(ctx,
		`INSERT INTO consents (farmer_id, satellite_monitoring)
		 VALUES ($1, $2)
		 ON CONFLICT (farmer_id) DO UPDATE
		   SET satellite_monitoring = EXCLUDED.satellite_monitoring,
		       consented_at = NOW()
		 RETURNING id, farmer_id, satellite_monitoring, consented_at`,
		farmerID, satellite,
	).Scan(&r.ID, &r.FarmerID, &r.SatelliteMonitoring, &consentedAt)
	if err != nil {
		return nil, wrap("create consent", err)
	}
	return &r, nil
}

// ── vi_reports (APPEND-ONLY — never UPDATE) ─────────────────────────────────

// VIReport is the dashboard projection of the latest report for a farm.
type VIReport struct {
	CVIMean         float64 `json:"cvi_mean"`
	NDVI            float64 `json:"ndvi"`
	ConfidenceScore float64 `json:"confidence_score"`
	PeriodStart     string  `json:"period_start"`
	PeriodEnd       string  `json:"period_end"`
}

// GetLatestVIReportPerFarm batches the lookup for every farm in one query,
// using DISTINCT ON — the strategy §10.10 says to preserve.
//
// Nothing currently writes to vi_reports (Appendix B Q5), so this returns an
// empty map in practice; the read path is ported anyway because /dashboard
// depends on it and the table will be populated in Phase 8.
func (s *Store) GetLatestVIReportPerFarm(ctx context.Context, farmIDs []string) (map[string]VIReport, error) {
	out := map[string]VIReport{}
	if len(farmIDs) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (farm_id)
		    farm_id, cvi_mean, ndvi, confidence_score, period_start, period_end
		 FROM vi_reports
		 WHERE farm_id = ANY($1::uuid[])
		 ORDER BY farm_id, created_at DESC`, farmIDs)
	if err != nil {
		return nil, wrap("get latest vi reports", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			farmID                 string
			cvi, ndvi, confidence  pgtype.Numeric
			periodStart, periodEnd pgtype.Date
		)
		if err := rows.Scan(&farmID, &cvi, &ndvi, &confidence, &periodStart, &periodEnd); err != nil {
			return nil, wrap("scan vi report", err)
		}
		cviF, err := numericToFloat(cvi)
		if err != nil {
			return nil, wrap("vi report cvi_mean", err)
		}
		ndviF, err := numericToFloat(ndvi)
		if err != nil {
			return nil, wrap("vi report ndvi", err)
		}
		confF, err := numericToFloat(confidence)
		if err != nil {
			return nil, wrap("vi report confidence_score", err)
		}
		out[farmID] = VIReport{
			CVIMean:         round(cviF, 4),
			NDVI:            round(ndviF, 4),
			ConfidenceScore: round(confF, 2),
			PeriodStart:     dateString(periodStart),
			PeriodEnd:       dateString(periodEnd),
		}
	}
	return out, wrap("iterate vi reports", rows.Err())
}
