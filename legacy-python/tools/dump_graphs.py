"""dump_graphs.py — Earth Engine expression-graph golden-fixture capture.

Serialises every EE computation the backend performs into JSON fixtures, so the
Go `eeexpr` builder can be tested for structural equality against the ground
truth produced by the real Python client.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §5.4, §5.7 (G1-G24).

Run inside the existing backend venv, from the backend directory:

    venv/Scripts/python.exe tools/dump_graphs.py

Writes to <repo>/internal/gee/eeexpr/testdata/.

Every dump() call here corresponds to exactly one row of §5.7 and one Go
builder function. The `dates` and geometry are pinned so the fixtures are
reproducible; the Go unit tests must be given the same pinned values.
"""

import datetime
import json
import pathlib
import sys

import ee
from ee import serializer

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1]))

import config  # noqa: E402
from services.gee_service import initialize_gee, _mask_clouds_scl  # noqa: E402
from services.index_service import compute_all_indices  # noqa: E402
from services.sar_service import _base_s1_collection, _speckle_filter  # noqa: E402
from services.radar_index_service import compute_radar_indices  # noqa: E402
from utils.geo_utils import geojson_to_ee_geometry  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
OUT = REPO / "internal" / "gee" / "eeexpr" / "testdata"
POLY = REPO / "testdata" / "polygons"

# Pinned so the fixture is reproducible. The Go builder must be given the same
# pinned dates in its unit tests — never datetime.date.today().
START, END = "2025-01-01", "2025-04-01"
DAY = "2025-02-14"
RADAR_DAY = "2025-02-14"

_written: list[str] = []


def dump(name: str, ee_object) -> None:
    graph = serializer.encode(ee_object, for_cloud_api=True)
    (OUT / f"{name}.json").write_text(
        json.dumps(graph, indent=2, sort_keys=True), encoding="utf-8"
    )
    _written.append(name)
    print("wrote", name)


def main() -> int:
    if not initialize_gee():
        print("GEE initialisation failed — cannot capture fixtures.", file=sys.stderr)
        return 1

    OUT.mkdir(parents=True, exist_ok=True)
    fixture_polygon = json.loads((POLY / "nashik_vineyard.json").read_text())
    geom = geojson_to_ee_geometry(fixture_polygon)
    dump("geometry", geom)

    # ── G1: S2 collection ────────────────────────────────────────────────────
    collection = (
        ee.ImageCollection(config.DATASET)
        .filterBounds(geom)
        .filterDate(START, END)
        .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", config.MAX_CLOUD_COVER_PCT))
        .map(_mask_clouds_scl)
        .map(lambda img: img.divide(10000))
    )
    dump("g01_s2_collection", collection)

    # ── G2 / G3: scene count + median composite ──────────────────────────────
    dump("g02_s2_scene_count", collection.size())
    composite = collection.median()
    dump("g03_s2_composite", composite)

    # ── G4: SCL cloud mask applied to a single image ─────────────────────────
    sample_img = ee.Image(config.DATASET + "/20250214T053001_20250214T053811_T43QCA")
    dump("g04_scl_mask", _mask_clouds_scl(sample_img))

    # ── G5: normalizedDifference indices ─────────────────────────────────────
    b = config.BANDS
    dump("g05_ndvi", composite.normalizedDifference([b["NIR"], b["RED"]]).rename("NDVI"))
    dump("g05_ndmi", composite.normalizedDifference([b["NIR"], b["SWIR"]]).rename("NDMI"))
    dump("g05_ndwi", composite.normalizedDifference([b["GREEN"], b["NIR"]]).rename("NDWI"))
    dump("g05_gndvi", composite.normalizedDifference([b["NIR"], b["GREEN"]]).rename("GNDVI"))

    # ── G6: EVI / SAVI — both the current expression() form and the explicit
    # band-arithmetic rewrite the Go port uses (PRD §7.3). Both are dumped so
    # the numeric-equivalence test has something to compare against.
    nir = composite.select(b["NIR"])
    red = composite.select(b["RED"])
    blue = composite.select(b["BLUE"])
    dump(
        "g06_evi_expression",
        composite.expression(
            "2.5 * (NIR - RED) / (NIR + 6.0 * RED - 7.5 * BLUE + 1.0)",
            {"NIR": nir, "RED": red, "BLUE": blue},
        ).rename("EVI"),
    )
    dump(
        "g06_evi_arith",
        nir.subtract(red)
        .multiply(2.5)
        .divide(nir.add(red.multiply(6.0)).subtract(blue.multiply(7.5)).add(1.0))
        .rename("EVI"),
    )
    dump(
        "g06_savi_expression",
        composite.expression(
            "((NIR - RED) / (NIR + RED + 0.5)) * 1.5",
            {"NIR": nir, "RED": red},
        ).rename("SAVI"),
    )
    dump(
        "g06_savi_arith",
        nir.subtract(red).divide(nir.add(red).add(0.5)).multiply(1.5).rename("SAVI"),
    )

    # ── G7: CVI weighted sum + full addBands stacking order ──────────────────
    indexed = compute_all_indices(composite)
    dump("g07_indexed", indexed)

    # ── G8: covering grid ────────────────────────────────────────────────────
    proj = ee.Projection("EPSG:4326").atScale(config.GRID_SCALE_M)
    grid = geom.coveringGrid(proj)
    dump("g08_grid", grid)
    dump("g08_grid_size", grid.size())

    # ── G9: per-cell reduction over the grid ─────────────────────────────────
    index_bands = ["NDVI", "EVI", "SAVI", "NDMI", "NDWI", "GNDVI", "CVI"]
    image_subset = indexed.select(index_bands)

    def _reduce_cell(cell):
        return cell.set(
            image_subset.reduceRegion(
                reducer=ee.Reducer.mean(),
                geometry=cell.geometry(),
                scale=config.GRID_SCALE_M,
                maxPixels=1e8,
            )
        )

    dump("g09_grid_reduced", grid.map(_reduce_cell))

    # ── G10 / G11: farm-wide mean + CVI stdDev ───────────────────────────────
    stats_bands = ["CVI", "NDVI", "EVI", "SAVI", "NDMI", "NDWI", "GNDVI"]
    dump(
        "g10_farm_mean",
        indexed.select(stats_bands).reduceRegion(
            reducer=ee.Reducer.mean(), geometry=geom, scale=10, maxPixels=1e9
        ),
    )
    dump(
        "g11_cvi_stddev",
        indexed.select(["CVI"]).reduceRegion(
            reducer=ee.Reducer.stdDev(), geometry=geom, scale=10, maxPixels=1e9
        ),
    )

    # ── G12: NDVI area histogram ─────────────────────────────────────────────
    ndvi = indexed.select("NDVI")
    bucket_img = ndvi.max(0.0).min(0.9999).divide(0.05).floor().int()
    dump(
        "g12_ndvi_histogram",
        ee.Image.pixelArea()
        .addBands(bucket_img)
        .reduceRegion(
            reducer=ee.Reducer.sum().group(groupField=1, groupName="bucket"),
            geometry=geom,
            scale=10,
            maxPixels=1e9,
        ),
    )

    # ── G13: smooth vegetation tile image ────────────────────────────────────
    dump(
        "g13_smooth_tile_ndvi",
        indexed.select("NDVI")
        .clip(geom)
        .updateMask(indexed.select("NDVI").gte(0))
        .resample("bicubic")
        .reproject(crs="EPSG:4326", scale=10)
        .focal_mean(2, "circle", "pixels"),
    )

    # ── G14: available S2 dates. NOTE: this builds its OWN collection —
    # filterBounds + filterDate + cloud filter only, with NO SCL mask and NO
    # /10000 scaling. Reusing `collection` above would not match the backend.
    dates_coll = (
        ee.ImageCollection(config.DATASET)
        .filterBounds(geom)
        .filterDate(START, END)
        .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", config.MAX_CLOUD_COVER_PCT))
    )
    dump(
        "g14_available_dates",
        dates_coll.map(
            lambda i: ee.Feature(
                None, {"date": ee.Date(i.get("system:time_start")).format("YYYY-MM-dd")}
            )
        )
        .aggregate_array("date")
        .distinct()
        .sort(),
    )

    # ── G15: single-day composite — window is [target, target+1) ─────────────
    target = datetime.date.fromisoformat(DAY)
    day_coll = (
        ee.ImageCollection(config.DATASET)
        .filterBounds(geom)
        .filterDate(target.isoformat(), (target + datetime.timedelta(days=1)).isoformat())
        .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", config.MAX_CLOUD_COVER_PCT))
        .map(_mask_clouds_scl)
        .map(lambda img: img.divide(10000))
    )
    dump("g15_single_day_collection", day_coll)
    dump("g15_single_day_composite", day_coll.median())

    # ── G16: point sample ────────────────────────────────────────────────────
    dump(
        "g16_point_sample",
        indexed.select("NDVI").reduceRegion(
            reducer=ee.Reducer.first(),
            geometry=ee.Geometry.Point([73.7908, 20.0116]),
            scale=10,
            maxPixels=1,
        ),
    )

    # ── G17: S1 base collection ──────────────────────────────────────────────
    s1_base = _base_s1_collection(geom)
    dump("g17_s1_base_collection", s1_base)

    # ── G18: speckle filter + composite window ───────────────────────────────
    rd = datetime.date.fromisoformat(RADAR_DAY)
    s1_start = (rd - datetime.timedelta(days=config.S1_DATE_WINDOW_DAYS)).isoformat()
    s1_end = (rd + datetime.timedelta(days=config.S1_DATE_WINDOW_DAYS + 1)).isoformat()
    s1_coll = s1_base.filterDate(s1_start, s1_end).map(_speckle_filter)
    dump("g18_s1_speckle_collection", s1_coll)
    dump("g18_s1_scene_count", s1_coll.size())
    s1_composite = s1_coll.median()
    dump("g18_s1_composite", s1_composite)

    # ── G19 / G20 / G21: radar indices ───────────────────────────────────────
    vv = s1_composite.select("VV")
    vh = s1_composite.select("VH")
    span = config.SMI_VV_WET_DB - config.SMI_VV_DRY_DB
    dump(
        "g19_smi",
        vv.subtract(config.SMI_VV_DRY_DB).divide(span).clamp(0.0, 1.0).rename("SMI"),
    )
    vv_lin = ee.Image(10.0).pow(vv.divide(10.0))
    vh_lin = ee.Image(10.0).pow(vh.divide(10.0))
    dump(
        "g20_rvi",
        vh_lin.multiply(4.0).divide(vv_lin.add(vh_lin)).clamp(0.0, 1.0).rename("RVI"),
    )
    dump("g21_ratio", vv.subtract(vh).rename("RATIO"))
    radar_image = compute_radar_indices(s1_composite)
    dump("g21_radar_indexed", radar_image)

    # ── G22: radar smooth tile image (no updateMask — dB is legitimately <0) ─
    dump(
        "g22_smooth_radar_tile_smi",
        radar_image.select("SMI")
        .clip(geom)
        .resample("bicubic")
        .reproject(crs="EPSG:4326", scale=10)
        .focal_mean(2, "circle", "pixels"),
    )

    # ── G23: radar per-cell reduction ────────────────────────────────────────
    radar_subset = radar_image.select(config.RADAR_BANDS)

    def _reduce_radar_cell(cell):
        return cell.set(
            radar_subset.reduceRegion(
                reducer=ee.Reducer.mean(),
                geometry=cell.geometry(),
                scale=config.GRID_SCALE_M,
                maxPixels=1e8,
            )
        )

    dump("g23_radar_grid_reduced", grid.map(_reduce_radar_cell))

    # ── G24: radar farm-wide mean ────────────────────────────────────────────
    dump(
        "g24_radar_farm_mean",
        radar_image.select(config.RADAR_BANDS).reduceRegion(
            reducer=ee.Reducer.mean(), geometry=geom, scale=10, maxPixels=1e9
        ),
    )

    # ── S1 available dates (backs /api/analyze-radar-dates) ──────────────────
    dump(
        "g17_s1_available_dates",
        s1_base.filterDate(START, END)
        .map(
            lambda i: ee.Feature(
                None, {"date": ee.Date(i.get("system:time_start")).format("YYYY-MM-dd")}
            )
        )
        .aggregate_array("date")
        .distinct()
        .sort(),
    )

    manifest = {
        "pinned": {"start": START, "end": END, "day": DAY, "radar_day": RADAR_DAY},
        "polygon": "testdata/polygons/nashik_vineyard.json",
        "fixtures": sorted(_written),
    }
    (OUT / "manifest.json").write_text(
        json.dumps(manifest, indent=2), encoding="utf-8"
    )
    print(f"\n{len(_written)} fixtures written to {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
