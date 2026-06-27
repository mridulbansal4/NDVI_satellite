-- =============================================================================
-- Satellite Agronomy Intelligence Platform — PostgreSQL + PostGIS Schema
-- Database: agri_platform
-- =============================================================================
-- HOW TO RUN:
--   psql -U postgres -d agri_platform -f schema.sql
--
-- NOTE: vi_reports is APPEND-ONLY. Never run UPDATE on this table.
--       All historical VI snapshots must be preserved for trend analysis.
-- =============================================================================

-- Extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUMS
-- =============================================================================

CREATE TYPE gender_enum AS ENUM ('male', 'female', 'other');

CREATE TYPE language_enum AS ENUM ('english', 'hindi', 'marathi', 'others');

CREATE TYPE area_unit_enum AS ENUM ('acres', 'hectares');

CREATE TYPE land_ownership_enum AS ENUM ('own_land', 'leased_land', 'contract_farming');

CREATE TYPE season_enum AS ENUM ('kharif', 'rabi', 'zaid');

CREATE TYPE irrigation_type_enum AS ENUM (
    'rainfed', 'borewell', 'canal', 'drip_irrigation', 'sprinkler'
);

CREATE TYPE soil_type_enum AS ENUM ('black', 'red', 'sandy', 'mixed', 'unknown');

-- =============================================================================
-- TABLE: farmers
-- Central identity record for each registered farmer.
-- =============================================================================
CREATE TABLE farmers (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    firebase_uid       VARCHAR(128) UNIQUE,
    mobile_number      VARCHAR(15)  UNIQUE NOT NULL,
    password_hash      VARCHAR(255),
    name               VARCHAR(255),
    age                INTEGER,
    gender             gender_enum,
    preferred_language language_enum NOT NULL DEFAULT 'english',
    created_at         TIMESTAMPTZ  DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  DEFAULT NOW()
);

-- =============================================================================
-- TABLE: farmer_locations
-- Primary address record for a farmer (village/PIN-code level).
-- =============================================================================
CREATE TABLE farmer_locations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id    UUID        NOT NULL REFERENCES farmers(id) ON DELETE CASCADE,
    pin_code     VARCHAR(6)  NOT NULL,
    state        VARCHAR(100),           -- auto-fetched via India Post PIN API
    district     VARCHAR(100),           -- auto-fetched via India Post PIN API
    taluka       VARCHAR(100),           -- auto-fetched via India Post PIN API
    village_name VARCHAR(255) NOT NULL,
    full_address TEXT
);

-- =============================================================================
-- TABLE: farms
-- Each farmer may own/operate multiple farm parcels.
-- boundary_geom is a PostGIS POLYGON in WGS84 (SRID 4326).
-- =============================================================================
CREATE TABLE farms (
    id                 UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id          UUID               NOT NULL REFERENCES farmers(id) ON DELETE CASCADE,
    farm_name          VARCHAR(255)       NOT NULL,
    total_area         NUMERIC(10,2)      NOT NULL,
    area_unit          area_unit_enum     NOT NULL,
    land_ownership     land_ownership_enum NOT NULL,
    latitude           DOUBLE PRECISION   NOT NULL,
    longitude          DOUBLE PRECISION   NOT NULL,
    boundary_geom      GEOMETRY(POLYGON, 4326),   -- PostGIS polygon boundary
    location_photo_url VARCHAR(1024),
    created_at         TIMESTAMPTZ        DEFAULT NOW()
);

-- =============================================================================
-- TABLE: crops
-- One row per crop sowing cycle on a farm.
-- =============================================================================
CREATE TABLE crops (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id                UUID         NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    crop_name              VARCHAR(255) NOT NULL,
    crop_variety           VARCHAR(255),
    sowing_date            DATE         NOT NULL,
    season                 season_enum  NOT NULL,
    expected_harvest_month VARCHAR(50),
    created_at             TIMESTAMPTZ  DEFAULT NOW()
);

-- =============================================================================
-- TABLE: irrigation
-- Irrigation method used for a farm parcel.
-- =============================================================================
CREATE TABLE irrigation (
    id              UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id         UUID                 NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    irrigation_type irrigation_type_enum NOT NULL,
    water_source    VARCHAR(255)
);

-- =============================================================================
-- TABLE: soil_info
-- OPTIONAL — one row per farm (UNIQUE constraint enforced).
-- May have no row if farmer skips soil step.
-- =============================================================================
CREATE TABLE soil_info (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id    UUID          UNIQUE REFERENCES farms(id) ON DELETE CASCADE,
    soil_type  soil_type_enum
);

-- =============================================================================
-- TABLE: consents
-- Farmer's consent to satellite monitoring (one consent record per farmer).
-- =============================================================================
CREATE TABLE consents (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id            UUID        UNIQUE NOT NULL REFERENCES farmers(id) ON DELETE CASCADE,
    satellite_monitoring BOOLEAN     NOT NULL,
    consented_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- TABLE: vi_reports
-- ⚠️  APPEND-ONLY — DO NOT UPDATE. One row per weekly VI Engine run.
-- Stores the composite vegetation index (CVI) and individual band indices.
-- Weights: NDVI=0.35, EVI=0.25, SAVI=0.15, NDMI=0.15, GNDVI=0.10
-- =============================================================================
CREATE TABLE vi_reports (
    id               UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id          UUID    NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    -- Composite Vegetation Index aggregates
    cvi_mean         FLOAT   NOT NULL,
    cvi_median       FLOAT   NOT NULL,
    cvi_std_dev      FLOAT   NOT NULL,
    -- Individual spectral indices
    ndvi             FLOAT   NOT NULL,  -- weight 0.35 — primary vegetation signal
    evi              FLOAT   NOT NULL,  -- weight 0.25 — enhanced vegetation (canopy)
    savi             FLOAT   NOT NULL,  -- weight 0.15 — soil-adjusted vegetation
    ndmi             FLOAT   NOT NULL,  -- weight 0.15 — moisture / drought stress
    ndwi             FLOAT   NOT NULL,  -- water body detection (unweighted)
    gndvi            FLOAT   NOT NULL,  -- weight 0.10 — chlorophyll / nutrient proxy
    -- Quality metadata
    confidence_score FLOAT   NOT NULL,  -- 0–100 scale
    scenes_used      INTEGER NOT NULL,
    period_start     DATE    NOT NULL,
    period_end       DATE    NOT NULL,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- farmers
CREATE INDEX idx_farmers_mobile_number  ON farmers(mobile_number);
CREATE INDEX idx_farmers_firebase_uid   ON farmers(firebase_uid);

-- farmer_locations
CREATE INDEX idx_farmer_locations_farmer_id ON farmer_locations(farmer_id);
CREATE INDEX idx_farmer_locations_pin_code  ON farmer_locations(pin_code);

-- farms
CREATE INDEX idx_farms_farmer_id ON farms(farmer_id);
CREATE INDEX idx_farms_boundary_geom ON farms USING GIST(boundary_geom);

-- crops
CREATE INDEX idx_crops_farm_id     ON crops(farm_id);
CREATE INDEX idx_crops_sowing_date ON crops(sowing_date);

-- vi_reports
CREATE INDEX idx_vi_reports_farm_id    ON vi_reports(farm_id);
CREATE INDEX idx_vi_reports_created_at ON vi_reports(created_at DESC);

-- =============================================================================
-- SEED DATA — 1 complete farmer chain
-- =============================================================================

DO $$
DECLARE
    v_farmer_id  UUID := gen_random_uuid();
    v_farm_id    UUID := gen_random_uuid();
BEGIN

    -- Farmer
    INSERT INTO farmers (id, firebase_uid, mobile_number, name, age, gender, preferred_language)
    VALUES (
        v_farmer_id,
        'firebase-uid-demo-001',
        '9876543210',
        'Ramesh Patil',
        42,
        'male',
        'marathi'
    );

    -- Farmer location (Nashik district, Maharashtra)
    INSERT INTO farmer_locations (farmer_id, pin_code, state, district, taluka, village_name, full_address)
    VALUES (
        v_farmer_id,
        '422001',
        'Maharashtra',
        'Nashik',
        'Nashik',
        'Pimpalgaon Baswant',
        'At Post Pimpalgaon Baswant, Tal. Nashik, Dist. Nashik, Maharashtra - 422001'
    );

    -- Farm
    INSERT INTO farms (id, farmer_id, farm_name, total_area, area_unit, land_ownership,
                       latitude, longitude, boundary_geom)
    VALUES (
        v_farm_id,
        v_farmer_id,
        'Ramesh''s Vineyard',
        2.5,
        'acres',
        'own_land',
        20.0116,
        73.7908,
        ST_GeomFromText(
            'POLYGON((73.7900 20.0110, 73.7916 20.0110, 73.7916 20.0122, 73.7900 20.0122, 73.7900 20.0110))',
            4326
        )
    );

    -- Crop (Kharif grapes — common in Nashik)
    INSERT INTO crops (farm_id, crop_name, crop_variety, sowing_date, season, expected_harvest_month)
    VALUES (
        v_farm_id,
        'Grapes',
        'Thompson Seedless',
        '2024-06-15',
        'kharif',
        'January'
    );

    -- Irrigation
    INSERT INTO irrigation (farm_id, irrigation_type, water_source)
    VALUES (
        v_farm_id,
        'drip_irrigation',
        'Borewell + Drip system, 2000 litre capacity tank'
    );

    -- Soil info
    INSERT INTO soil_info (farm_id, soil_type)
    VALUES (
        v_farm_id,
        'black'
    );

    -- Consent
    INSERT INTO consents (farmer_id, satellite_monitoring)
    VALUES (
        v_farmer_id,
        TRUE
    );

    -- VI Report (realistic Sentinel-2 derived values for a healthy vineyard)
    INSERT INTO vi_reports (
        farm_id,
        cvi_mean, cvi_median, cvi_std_dev,
        ndvi, evi, savi, ndmi, ndwi, gndvi,
        confidence_score, scenes_used,
        period_start, period_end
    )
    VALUES (
        v_farm_id,
        0.68, 0.71, 0.09,
        0.74, 0.62, 0.58, 0.41, 0.12, 0.65,
        87.4, 6,
        '2025-04-15', '2025-04-22'
    );

END $$;
