package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/SanTiwari07/NDVI_satellite/internal/repo"
)

// Dashboard is the GET /dashboard payload (§10.10).
//
// Field types here are the ones that "bite in Go":
//   - TotalArea is float64 from a NUMERIC(10,2), so it emits 2.5 not "2.50"
//   - BoundaryGeom is raw JSON and is null when no polygon was stored
//   - SowingDate / PeriodStart / PeriodEnd are "YYYY-MM-DD" strings, not RFC3339
//   - LatestVIReport is a pointer so it emits null when the farm has no report
type Dashboard struct {
	Farmer DashboardFarmer `json:"farmer"`
	Farms  []DashboardFarm `json:"farms"`
}

// DashboardFarmer is the farmer profile block.
type DashboardFarmer struct {
	ID                string  `json:"id"`
	Name              *string `json:"name"`
	MobileNumber      string  `json:"mobile_number"`
	PreferredLanguage *string `json:"preferred_language"`
}

// DashboardFarm is one farm with its crops and latest report.
type DashboardFarm struct {
	ID             string          `json:"id"`
	FarmName       string          `json:"farm_name"`
	TotalArea      float64         `json:"total_area"`
	AreaUnit       string          `json:"area_unit"`
	Latitude       float64         `json:"latitude"`
	Longitude      float64         `json:"longitude"`
	BoundaryGeom   json.RawMessage `json:"boundary_geom"`
	Crops          []repo.Crop     `json:"crops"`
	LatestVIReport *repo.VIReport  `json:"latest_vi_report"`
}

// ErrFarmerNotFound maps to the 404 of §10.10.
var ErrFarmerNotFound = errors.New("farmer not found")

// GetDashboard assembles the dashboard payload.
//
// Query strategy is preserved verbatim (§10.10): one farms query, ONE batched
// vi_reports query using DISTINCT ON … WHERE farm_id = ANY($1::uuid[]), then
// one crops query PER FARM. That last one is an N+1 and is deliberate — it is
// filed as K7 and batching it is a Phase 8 change.
func (o *Onboarding) GetDashboard(ctx context.Context, farmerID string) (*Dashboard, error) {
	farmer, err := o.Store.FindFarmerByID(ctx, farmerID)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, fmt.Errorf("%w: Farmer %s not found.", ErrFarmerNotFound, farmerID)
	}
	if err != nil {
		return nil, err
	}

	farms, err := o.Store.GetFarmsByFarmer(ctx, farmerID)
	if err != nil {
		return nil, err
	}

	farmIDs := make([]string, 0, len(farms))
	for _, f := range farms {
		farmIDs = append(farmIDs, f.ID)
	}
	viMap, err := o.Store.GetLatestVIReportPerFarm(ctx, farmIDs)
	if err != nil {
		return nil, err
	}

	assembled := make([]DashboardFarm, 0, len(farms))
	for _, f := range farms {
		crops, err := o.Store.GetCropsByFarm(ctx, f.ID) // K7: N+1, preserved
		if err != nil {
			return nil, err
		}

		df := DashboardFarm{
			ID:           f.ID,
			FarmName:     f.FarmName,
			TotalArea:    f.TotalArea,
			AreaUnit:     f.AreaUnit,
			Latitude:     f.Latitude,
			Longitude:    f.Longitude,
			BoundaryGeom: f.BoundaryGeom,
			Crops:        crops,
		}
		if vi, ok := viMap[f.ID]; ok {
			v := vi
			df.LatestVIReport = &v
		}
		assembled = append(assembled, df)
	}

	o.Log.Info(fmt.Sprintf("[Dashboard] Assembled dashboard for farmer %s (%d farms)",
		farmerID, len(farms)))

	return &Dashboard{
		Farmer: DashboardFarmer{
			ID:                farmer.ID,
			Name:              farmer.Name,
			MobileNumber:      farmer.MobileNumber,
			PreferredLanguage: farmer.PreferredLanguage,
		},
		Farms: assembled,
	}, nil
}
