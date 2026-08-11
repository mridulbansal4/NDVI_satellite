"""
services/radar_grid_service.py — Radar Grid Reduction + Zonal Stats
===================================================================
The Sentinel-1 counterpart of the reduction half of grid_service.py.

Responsibilities:
    - Reduce mean radar band values (VV, VH, SMI, RVI, RATIO) per grid cell
    - Attach a moisture classification label (Dry / Moderate / Wet) per cell
    - Apply the shared Gaussian spatial smoothing for a continuous heatmap
    - Compute farm-wide radar zonal statistics for the sidebar

Grid generation itself is reused from grid_service.generate_grid() — the tiling
is geometry-only and identical for both pipelines. Only the reduction differs,
so it lives here to keep the vegetation reducer untouched.
"""

import logging
import ee

from config import GRID_SCALE_M, RADAR_BANDS, SMI_THRESHOLDS
from services.grid_service import _smooth_grid_values

logger = logging.getLogger(__name__)


# ─────────────────────────────────────────────────────────────────────────────
# Interpretation
# ─────────────────────────────────────────────────────────────────────────────

def classify_moisture(smi: float | None) -> str:
    """Map an SMI value (0–1) to Dry / Moderate / Wet (thresholds in config)."""
    if smi is None:
        return "No data"
    for threshold in sorted(SMI_THRESHOLDS.keys(), reverse=True):
        if smi >= threshold:
            return SMI_THRESHOLDS[threshold]
    return "Dry"


# ─────────────────────────────────────────────────────────────────────────────
# Grid Reduction
# ─────────────────────────────────────────────────────────────────────────────

def reduce_radar_grid_values(
    radar_image: ee.Image,
    grid: ee.FeatureCollection,
    ee_geometry: ee.Geometry,
    scale: int = GRID_SCALE_M,
) -> dict:
    """
    Reduce mean radar band values for each grid cell and return as GeoJSON.

    For each cell: mean VV, VH, SMI, RVI, RATIO → rounded → moisture class.
    SMI is spatially smoothed (like the vegetation grid) for a continuous look.

    Returns a GeoJSON FeatureCollection dict ready for jsonify().
    """
    image_subset = radar_image.select(RADAR_BANDS)

    def _reduce_cell(cell: ee.Feature) -> ee.Feature:
        stats = image_subset.reduceRegion(
            reducer=ee.Reducer.mean(),
            geometry=cell.geometry(),
            scale=scale,
            maxPixels=1e8,
        )
        return cell.set(stats)

    reduced = grid.map(_reduce_cell)

    logger.info("Reducing radar values per grid cell…")
    raw_geojson = reduced.getInfo()   # triggers GEE computation

    features = []
    for feature in raw_geojson.get("features", []):
        props = feature.get("properties", {})
        rounded = {}
        for band in RADAR_BANDS:
            val = props.get(band)
            rounded[band.lower()] = round(val, 4) if val is not None else None
        features.append({
            "type": "Feature",
            "geometry": feature["geometry"],
            "properties": rounded,
        })

    # Smooth the moisture-oriented bands so the heatmap reads continuously.
    features = _smooth_grid_values(
        features, [b.lower() for b in RADAR_BANDS], sigma_factor=0.6
    )

    # (Re)attach moisture classification after smoothing.
    for feat in features:
        feat["properties"]["moisture_classification"] = classify_moisture(
            feat["properties"].get("smi")
        )

    logger.info("Radar grid reduction complete: %d features (smoothed).", len(features))
    return {"type": "FeatureCollection", "features": features}


# ─────────────────────────────────────────────────────────────────────────────
# Farm-wide Radar Statistics
# ─────────────────────────────────────────────────────────────────────────────

def extract_radar_statistics(
    radar_image: ee.Image,
    ee_geometry: ee.Geometry,
    scene_count: int,
) -> dict:
    """
    Compute farm-wide mean radar statistics for the sidebar:
        vv_mean, vh_mean, vv_vh_ratio, smi, moisture_classification.
    """
    try:
        mean_result = radar_image.select(RADAR_BANDS).reduceRegion(
            reducer=ee.Reducer.mean(),
            geometry=ee_geometry,
            scale=10,
            maxPixels=1e9,
        ).getInfo()
    except Exception as exc:
        logger.error("Failed to extract radar stats: %s", exc)
        mean_result = {b: None for b in RADAR_BANDS}

    def _r(key):
        v = mean_result.get(key)
        return round(v, 4) if v is not None else None

    smi = _r("SMI")
    return {
        "scene_count":            scene_count,
        "vv_mean":                _r("VV"),
        "vh_mean":                _r("VH"),
        "vv_vh_ratio":            _r("RATIO"),
        "rvi":                    _r("RVI"),
        "smi":                    smi,
        "moisture_classification": classify_moisture(smi),
    }
