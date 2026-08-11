"""
services/sar_service.py — Sentinel-1 (SAR) Data Layer
======================================================
The Sentinel-1 counterpart of gee_service.py.

Responsibilities:
    - Fetch and pre-process Sentinel-1 GRD imagery for a given polygon
    - Apply a light speckle filter to the VV/VH backscatter (dB) bands
    - Produce a median composite for a target date window
    - List available S1 acquisition dates
    - Generate smooth radar tile URLs

This is an INDEPENDENT pipeline from the Sentinel-2 vegetation path. It reuses
the shared GEE session initialised by gee_service.initialize_gee(), but is the
only module (besides gee_service) that talks to GEE directly. Everything here
returns ee.Image / ee.Geometry objects for the radar_* service modules to consume.

Sentinel-1 GRD notes:
    - VV / VH bands are already log-scaled (dB) — no ÷10000 scaling like S2.
    - We pin one orbit pass (DESCENDING) so backscatter is comparable over time.
"""

import logging
import datetime
import ee

from config import (
    S1_DATASET,
    S1_INSTRUMENT_MODE,
    S1_ORBIT_PASS,
    S1_RESOLUTION_M,
    S1_LOOKBACK_DAYS,
    S1_DATE_WINDOW_DAYS,
    S1_SPECKLE_RADIUS_M,
    S1_SPECKLE_KERNEL,
)

logger = logging.getLogger(__name__)


# ─────────────────────────────────────────────────────────────────────────────
# Collection filtering
# ─────────────────────────────────────────────────────────────────────────────

def _base_s1_collection(ee_geometry: ee.Geometry) -> ee.ImageCollection:
    """
    Build the base Sentinel-1 collection filtered by mode, polarisation,
    resolution and orbit pass — but NOT yet by date. Selects VV + VH only.
    """
    return (
        ee.ImageCollection(S1_DATASET)
        .filterBounds(ee_geometry)
        .filter(ee.Filter.eq("instrumentMode", S1_INSTRUMENT_MODE))
        .filter(ee.Filter.listContains("transmitterReceiverPolarisation", "VV"))
        .filter(ee.Filter.listContains("transmitterReceiverPolarisation", "VH"))
        .filter(ee.Filter.eq("resolution_meters", S1_RESOLUTION_M))
        .filter(ee.Filter.eq("orbitProperties_pass", S1_ORBIT_PASS))
        .select(["VV", "VH"])
    )


def _speckle_filter(image: ee.Image) -> ee.Image:
    """
    Apply a light spatial speckle filter (focal median) to the VV/VH dB bands.

    SAR imagery is inherently noisy (speckle). A small focal-median smooth
    (radius ~50 m) suppresses that noise before we derive indices, without
    materially blurring field-scale structure.
    """
    smoothed = image.focal_median(
        radius=S1_SPECKLE_RADIUS_M,
        kernelType=S1_SPECKLE_KERNEL,
        units="meters",
    )
    # Preserve band names + image metadata (time_start) for downstream use.
    return smoothed.rename(["VV", "VH"]).copyProperties(image, ["system:time_start"])


# ─────────────────────────────────────────────────────────────────────────────
# Public API
# ─────────────────────────────────────────────────────────────────────────────

def get_s1_composite(
    ee_geometry: ee.Geometry,
    target_date: str | None = None,
    window_days: int = S1_DATE_WINDOW_DAYS,
) -> tuple[ee.Image | None, int, str, str]:
    """
    Fetch a speckle-filtered Sentinel-1 median composite for a target date.

    Pipeline:
        1. Filter the S1 GRD collection (mode/pol/resolution/orbit/bounds).
        2. Restrict to [target_date - window, target_date + window].
           If no target_date is given, use the most recent `window` window.
        3. Speckle-filter every scene (focal median).
        4. Reduce to a median composite (further reduces speckle).

    Args:
        ee_geometry: Farm polygon geometry.
        target_date: ISO date string 'YYYY-MM-DD', or None for latest.
        window_days: ± days around target_date to include.

    Returns:
        (composite | None, scene_count, start_str, end_str)
        composite is None when no scenes are found.
    """
    if target_date:
        center = datetime.date.fromisoformat(target_date)
        start = center - datetime.timedelta(days=window_days)
        end   = center + datetime.timedelta(days=window_days + 1)  # end exclusive
    else:
        end = datetime.date.today()
        start = end - datetime.timedelta(days=window_days * 2)
        end = end + datetime.timedelta(days=1)

    start_str, end_str = start.isoformat(), end.isoformat()

    logger.info(
        "Fetching S1 composite | %s → %s | mode=%s | orbit=%s",
        start_str, end_str, S1_INSTRUMENT_MODE, S1_ORBIT_PASS,
    )

    collection = (
        _base_s1_collection(ee_geometry)
        .filterDate(start_str, end_str)
        .map(_speckle_filter)
    )

    scene_count = collection.size().getInfo()
    logger.info("S1 scenes found after filtering: %d", scene_count)

    if scene_count == 0:
        logger.warning(
            "No Sentinel-1 scenes for this window. "
            "Try a different date or widen S1_DATE_WINDOW_DAYS in config.py."
        )
        return None, 0, start_str, end_str

    composite = collection.median()
    return composite, scene_count, start_str, end_str


def get_s1_available_dates(
    ee_geometry: ee.Geometry,
    lookback_days: int = S1_LOOKBACK_DAYS,
) -> list[str]:
    """
    Return a sorted list of unique Sentinel-1 acquisition dates (YYYY-MM-DD)
    covering the geometry over the last `lookback_days`, for the configured
    mode / polarisation / orbit pass.
    """
    end_date = datetime.date.today()
    start_date = end_date - datetime.timedelta(days=lookback_days)

    collection = (
        _base_s1_collection(ee_geometry)
        .filterDate(start_date.isoformat(), end_date.isoformat())
    )

    def _get_date(img):
        d = ee.Date(img.get("system:time_start")).format("YYYY-MM-dd")
        return ee.Feature(None, {"date": d})

    date_list = (
        collection.map(_get_date)
        .aggregate_array("date")
        .distinct()
        .sort()
        .getInfo()
    )

    logger.info("Available S1 dates (%s → %s): %d unique",
                start_date.isoformat(), end_date.isoformat(), len(date_list))
    return date_list


def get_smooth_radar_tile_url(
    image: ee.Image,
    ee_geometry: ee.Geometry,
    band: str,
    vis_params: dict,
) -> str | None:
    """
    Generate a smooth, bicubic-resampled GEE tile URL for a single radar band,
    clipped to the farm polygon. Mirrors gee_service.get_smooth_tile_url but
    without the vegetation-specific negative-value masking (radar dB values are
    legitimately negative).

    Args:
        image      : Multi-band radar ee.Image (VV, VH, SMI, RVI, RATIO).
        ee_geometry: Farm polygon geometry for clipping.
        band       : Band name (e.g. 'SMI', 'VV').
        vis_params : Dict with 'min', 'max', 'palette' keys.

    Returns:
        Tile URL string or None on failure.
    """
    try:
        smooth_image = (
            image
            .select(band)
            .clip(ee_geometry)
            .resample("bicubic")
            .reproject(crs="EPSG:4326", scale=10)
            .focal_mean(2, "circle", "pixels")
        )
        map_id_dict = smooth_image.getMapId(vis_params)
        url = map_id_dict["tile_fetcher"].url_format
        logger.info("Smooth radar tile URL generated for band=%s", band)
        return url
    except Exception as exc:
        logger.error("Failed to generate radar tile URL for %s: %s", band, exc)
        return None
