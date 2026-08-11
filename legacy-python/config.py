"""
config.py — Central Configuration for CVI Engine Web App
==========================================================
All GEE settings, spatial parameters, index weights, CVI thresholds,
and grid parameters are centralized here.

Edit this file to tune the engine — no changes to business logic needed.
"""

import os
from dotenv import load_dotenv

load_dotenv()

# ─────────────────────────────────────────────────────────────────────────────
# Google Earth Engine
# ─────────────────────────────────────────────────────────────────────────────
GEE_PROJECT_ID = os.getenv("GEE_PROJECT_ID")
DATASET = "COPERNICUS/S2_SR_HARMONIZED"   # Sentinel-2 Surface Reflectance
# ─────────────────────────────────────────────────────────────────────────────
# Temporal Window
# ─────────────────────────────────────────────────────────────────────────────
# Number of days to look back from today for satellite imagery
LOOKBACK_DAYS = 90   # 3 months

# ─────────────────────────────────────────────────────────────────────────────
# Cloud Filtering
# ─────────────────────────────────────────────────────────────────────────────
MAX_CLOUD_COVER_PCT = 20  # Relaxed slightly for polygon analysis

# SCL band values to mask (per-pixel cloud/shadow removal)
# 3=Cloud Shadow, 8=Medium Cloud, 9=High Cloud, 10=Cirrus
SCL_MASK_VALUES = [3, 8, 9, 10]

# ─────────────────────────────────────────────────────────────────────────────
# Sentinel-2 Band Aliases
# ─────────────────────────────────────────────────────────────────────────────
BANDS = {
    "BLUE":  "B2",    # ~490 nm
    "GREEN": "B3",    # ~560 nm
    "RED":   "B4",    # ~665 nm
    "NIR":   "B8",    # ~842 nm  (10 m native)
    "SWIR":  "B11",   # ~1610 nm (20 m — resampled by GEE on-the-fly)
    "SCL":   "SCL",   # Scene Classification Layer
}

# ─────────────────────────────────────────────────────────────────────────────
# Composite Vegetation Index (CVI) Weights — must sum to 1.0
# ─────────────────────────────────────────────────────────────────────────────
CVI_WEIGHTS = {
    "NDVI":  0.70,   # Primary signal - highly robust against terrain shadows
    "EVI":   0.10,   # Nominal shadow-sensitive weight
    "SAVI":  0.05,   # Nominal shadow-sensitive weight
    "NDMI":  0.10,   # Moisture intelligence
    "GNDVI": 0.05,   # Chlorophyll signal
}

# ─────────────────────────────────────────────────────────────────────────────
# Grid Parameters
# ─────────────────────────────────────────────────────────────────────────────
GRID_SCALE_M = 10          # High-resolution 10m grid (Sentinel-2 native)
MAX_GRID_CELLS = 2000       # Allow up to 2000 cells for fine-resolution coverage
GRID_SCALE_STEP_M = 2      # Increment in 2m steps when auto-scaling up

# ─────────────────────────────────────────────────────────────────────────────
# CVI Interpretation Thresholds
# ─────────────────────────────────────────────────────────────────────────────
CVI_THRESHOLDS = {
    0.5:  "Healthy vegetation",
    0.25: "Moderate vegetation, possible stress",
    -1.0: "Poor vegetation, needs attention",
}

# ─────────────────────────────────────────────────────────────────────────────
# Individual Index Interpretation Thresholds
# ─────────────────────────────────────────────────────────────────────────────
NDVI_THRESHOLDS = {
    0.6:  "Dense, healthy vegetation",
    0.4:  "Moderate vegetation",
    0.2:  "Sparse / stressed vegetation",
    0.0:  "Bare soil",
    -1.0: "Water / non-vegetated",
}
EVI_THRESHOLDS = {
    0.5:  "Dense vegetation",
    0.3:  "Moderate vegetation",
    0.1:  "Sparse vegetation",
    -1.0: "Bare / non-vegetated",
}
SAVI_THRESHOLDS = {
    0.5:  "High vegetation + low soil effect",
    0.3:  "Moderate vegetation",
    0.1:  "Low vegetation",
    -1.0: "Bare soil dominant",
}
NDMI_THRESHOLDS = {
    0.4:  "High moisture",
    0.2:  "Moderate moisture",
    0.0:  "Low moisture",
    -1.0: "Dry / drought stress",
}
NDWI_THRESHOLDS = {
    0.3:  "High water presence",
    0.0:  "Water likely present",
    -1.0: "No significant water",
}
GNDVI_THRESHOLDS = {
    0.6:  "Excellent chlorophyll / nutrient status",
    0.4:  "Good chlorophyll",
    0.2:  "Moderate",
    -1.0: "Low chlorophyll",
}

# ─────────────────────────────────────────────────────────────────────────────
# Sentinel-1 (SAR) — Soil Moisture / Radar Module
# ─────────────────────────────────────────────────────────────────────────────
# This is a SEPARATE, independent pipeline from the Sentinel-2 optical config
# above. It powers the radar soil-moisture layers (SMI, RVI, VV/VH ratio, VV, VH)
# and never touches the vegetation index path.
S1_DATASET            = "COPERNICUS/S1_GRD"   # Sentinel-1 Ground Range Detected
S1_INSTRUMENT_MODE    = "IW"                   # Interferometric Wide swath
S1_ORBIT_PASS         = "DESCENDING"           # keep one pass so dB is comparable over time
S1_RESOLUTION_M       = 10                      # resolution_meters filter
S1_LOOKBACK_DAYS      = 90                      # window for listing available S1 dates
S1_DATE_WINDOW_DAYS   = 6                        # ± days around the selected date for the median composite

# Speckle filter (applied to VV/VH dB bands before computing indices).
S1_SPECKLE_RADIUS_M   = 50                       # focal_median radius in metres
S1_SPECKLE_KERNEL     = "circle"

# Soil Moisture Index (SMI) calibration bounds, in dB of VV backscatter.
# Wetter soil returns a stronger (less negative) signal. These are fixed
# reference bounds so the index is comparable across dates — tune here.
SMI_VV_DRY_DB         = -20.0                    # dry-soil reference (→ SMI 0)
SMI_VV_WET_DB         = -8.0                     # wet-soil reference (→ SMI 1)

# Radar layers exposed to the frontend dropdown (lowercase keys = grid props).
RADAR_BANDS = ["SMI", "RVI", "RATIO", "VV", "VH"]

# Per-band normalisation bounds used purely for colour mapping / tiles.
# (The raw values in the grid + sidebar are the true physical values.)
RADAR_VIS_BOUNDS = {
    "SMI":   {"min": 0.0,   "max": 1.0},    # already 0–1
    "RVI":   {"min": 0.0,   "max": 1.0},    # linear-power ratio, ~0–1
    "RATIO": {"min": 2.0,   "max": 16.0},   # VV−VH in dB
    "VV":    {"min": -22.0, "max": -6.0},   # VV backscatter dB
    "VH":    {"min": -28.0, "max": -12.0},  # VH backscatter dB
}

# Moisture classification thresholds on the SMI (0–1) scale.
SMI_THRESHOLDS = {
    0.66: "Wet",
    0.33: "Moderate",
    0.0:  "Dry",
}

# ─────────────────────────────────────────────────────────────────────────────
# Confidence Score Parameters
# ─────────────────────────────────────────────────────────────────────────────
CONFIDENCE_SCENE_TARGET = 5
CONFIDENCE_STD_MAX = 0.3

# ─────────────────────────────────────────────────────────────────────────────
# Logging
# ─────────────────────────────────────────────────────────────────────────────
LOG_LEVEL  = "INFO"
LOG_FORMAT = "%(asctime)s [%(levelname)s] %(name)s — %(message)s"
LOG_DATE   = "%Y-%m-%d %H:%M:%S"
LOG_FILE   = "cvi_engine.log"
