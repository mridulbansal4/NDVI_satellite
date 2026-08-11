package repo

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// FarmRow is the INSERT … RETURNING row POST /farm echoes (§10.10).
type FarmRow struct {
	ID       string `json:"id"`
	FarmerID string `json:"farmer_id"`
	FarmName string `json:"farm_name"`
	// TotalArea is a STRING here — psycopg2 maps NUMERIC to Decimal and Flask
	// serialises that as "2.50". /dashboard emits the same column as the number
	// 2.5 because the Python casts it with float() there. See numericToString.
	TotalArea     string  `json:"total_area"`
	AreaUnit      string  `json:"area_unit"`
	LandOwnership string  `json:"land_ownership"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	CreatedAt     string  `json:"created_at"`
}

// Farm is a dashboard row, including the boundary as raw GeoJSON.
type Farm struct {
	ID            string
	FarmerID      string
	FarmName      string
	TotalArea     float64
	AreaUnit      string
	LandOwnership string
	Latitude      float64
	Longitude     float64
	// BoundaryGeom is ST_AsGeoJSON(...)::json and is NULL when no polygon was
	// stored, so it must serialise as JSON null rather than {} (§10.10).
	BoundaryGeom json.RawMessage
}

// CreateFarm inserts a farm, converting the GeoJSON boundary via PostGIS.
//
// Two statements rather than one with a conditional, mirroring the Python:
// ST_GeomFromGeoJSON(NULL) is an error, so the column is omitted entirely when
// no boundary was supplied.
func (s *Store) CreateFarm(
	ctx context.Context,
	farmerID, farmName string, totalArea float64, areaUnit, landOwnership string,
	latitude, longitude float64, boundary json.RawMessage, photoURL *string,
) (*FarmRow, error) {
	var (
		f         FarmRow
		area      pgtype.Numeric
		createdAt pgtype.Timestamptz
	)

	scan := func(row interface {
		Scan(dest ...any) error
	}) error {
		return row.Scan(&f.ID, &f.FarmerID, &f.FarmName, &area, &f.AreaUnit,
			&f.LandOwnership, &f.Latitude, &f.Longitude, &createdAt)
	}

	var err error
	if len(boundary) > 0 && string(boundary) != "null" {
		err = scan(s.pool.QueryRow(ctx,
			`INSERT INTO farms
			   (farmer_id, farm_name, total_area, area_unit, land_ownership,
			    latitude, longitude, boundary_geom, location_photo_url)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, ST_GeomFromGeoJSON($8), $9)
			 RETURNING id, farmer_id, farm_name, total_area, area_unit,
			           land_ownership, latitude, longitude, created_at`,
			farmerID, farmName, totalArea, areaUnit, landOwnership,
			latitude, longitude, string(boundary), photoURL))
	} else {
		err = scan(s.pool.QueryRow(ctx,
			`INSERT INTO farms
			   (farmer_id, farm_name, total_area, area_unit, land_ownership,
			    latitude, longitude, location_photo_url)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING id, farmer_id, farm_name, total_area, area_unit,
			           land_ownership, latitude, longitude, created_at`,
			farmerID, farmName, totalArea, areaUnit, landOwnership,
			latitude, longitude, photoURL))
	}
	if err != nil {
		return nil, wrap("create farm", err)
	}

	f.TotalArea = numericToString(area)
	if createdAt.Valid {
		// Python's jsonify renders a datetime as an RFC 1123 string.
		f.CreatedAt = createdAt.Time.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	}
	return &f, nil
}

// GetFarmsByFarmer returns every farm for a farmer, oldest first.
func (s *Store) GetFarmsByFarmer(ctx context.Context, farmerID string) ([]Farm, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, farm_name, total_area, area_unit, land_ownership,
		        latitude, longitude, ST_AsGeoJSON(boundary_geom)::json AS boundary_geom
		 FROM farms WHERE farmer_id = $1 ORDER BY created_at ASC`, farmerID)
	if err != nil {
		return nil, wrap("get farms by farmer", err)
	}
	defer rows.Close()

	out := []Farm{}
	for rows.Next() {
		var (
			f    Farm
			area pgtype.Numeric
		)
		if err := rows.Scan(&f.ID, &f.FarmName, &area, &f.AreaUnit,
			&f.LandOwnership, &f.Latitude, &f.Longitude, &f.BoundaryGeom); err != nil {
			return nil, wrap("scan farm", err)
		}
		if f.TotalArea, err = numericToFloat(area); err != nil {
			return nil, wrap("farm total_area", err)
		}
		f.FarmerID = farmerID
		out = append(out, f)
	}
	return out, wrap("iterate farms", rows.Err())
}

// GetFarmByID returns one farm, used for the ownership check.
func (s *Store) GetFarmByID(ctx context.Context, farmID string) (*Farm, error) {
	var (
		f    Farm
		area pgtype.Numeric
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, farmer_id, farm_name, total_area, area_unit, land_ownership,
		        latitude, longitude, ST_AsGeoJSON(boundary_geom)::json AS boundary_geom
		 FROM farms WHERE id = $1`, farmID,
	).Scan(&f.ID, &f.FarmerID, &f.FarmName, &area, &f.AreaUnit,
		&f.LandOwnership, &f.Latitude, &f.Longitude, &f.BoundaryGeom)
	if err != nil {
		return nil, wrap("get farm by id", err)
	}
	if f.TotalArea, err = numericToFloat(area); err != nil {
		return nil, wrap("farm total_area", err)
	}
	return &f, nil
}
