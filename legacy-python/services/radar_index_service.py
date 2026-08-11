"""
services/radar_index_service.py — Sentinel-1 Radar Index Computation
=====================================================================
The Sentinel-1 counterpart of index_service.py. Computes the radar soil /
vegetation layers from a speckle-filtered VV/VH (dB) composite.

Layers (bands) produced:
    VV     — raw VV backscatter (dB). Sensitive to surface soil moisture + roughness.
    VH     — raw VH backscatter (dB). Sensitive to vegetation structure.
    SMI    — Soil Moisture Index, VV normalised to 0–1 against dry/wet dB bounds.
    RVI    — Radar Vegetation Index = 4*VH_lin / (VV_lin + VH_lin) (linear power).
    RATIO  — VV_dB − VH_dB (dB subtraction = linear division).

All operate lazily on GEE. The multi-band image returned is consumed by
radar_grid_service for per-cell reduction and by sar_service for tiles.
"""

import logging
import ee

from config import SMI_VV_DRY_DB, SMI_VV_WET_DB

logger = logging.getLogger(__name__)


def _compute_smi(vv_db: ee.Image) -> ee.Image:
    """
    SMI = (VV − VV_dry) / (VV_wet − VV_dry), clamped to [0, 1].

    Wetter soil produces a stronger (less negative) VV return, so higher SMI
    means wetter soil. Uses fixed dB calibration bounds from config so the
    index is comparable across dates.
    """
    span = SMI_VV_WET_DB - SMI_VV_DRY_DB
    return (
        vv_db.subtract(SMI_VV_DRY_DB)
        .divide(span)
        .clamp(0.0, 1.0)
        .rename("SMI")
    )


def _compute_rvi(vv_db: ee.Image, vh_db: ee.Image) -> ee.Image:
    """
    RVI = 4 * VH_linear / (VV_linear + VH_linear).

    Backscatter must be converted from dB to linear power first:
        linear = 10 ^ (dB / 10)
    Higher RVI = denser / healthier canopy. Clamped to [0, 1] for stability.
    """
    vv_lin = ee.Image(10.0).pow(vv_db.divide(10.0))
    vh_lin = ee.Image(10.0).pow(vh_db.divide(10.0))
    return (
        vh_lin.multiply(4.0)
        .divide(vv_lin.add(vh_lin))
        .clamp(0.0, 1.0)
        .rename("RVI")
    )


def _compute_ratio(vv_db: ee.Image, vh_db: ee.Image) -> ee.Image:
    """
    VV/VH ratio in dB = VV_dB − VH_dB (subtraction in dB == division in linear).
    High ratio → soil dominates (wet / bare). Low ratio → vegetation dominates.
    """
    return vv_db.subtract(vh_db).rename("RATIO")


def compute_radar_indices(composite: ee.Image) -> ee.Image:
    """
    Compute all radar layers from a speckle-filtered S1 composite.

    Input:
        composite: ee.Image with VV, VH bands in dB (speckle-filtered).

    Output:
        ee.Image with bands: VV, VH, SMI, RVI, RATIO
    """
    vv = composite.select("VV")
    vh = composite.select("VH")

    smi   = _compute_smi(vv)
    rvi   = _compute_rvi(vv, vh)
    ratio = _compute_ratio(vv, vh)

    indexed = composite.select(["VV", "VH"]).addBands([smi, rvi, ratio])
    logger.info("Radar indices computed: VV, VH, SMI, RVI, RATIO")
    return indexed
