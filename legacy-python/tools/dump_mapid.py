"""dump_mapid.py — Earth Engine `maps` request/response golden fixture.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §5.2 and §5.6 (both marked VERIFY).

Captures, from a real live call:
  * the exact JSON body the Python client POSTs to projects/{p}/maps
    (intercepted at the googleapiclient layer, so it is the true wire format)
  * the returned resource name and the tile URL template the backend hands to
    the frontend as `ndvi_tile_url`

The Go CreateMap builder must reproduce the request body and the URL template
character for character.

Run from the backend directory:
    venv/Scripts/python.exe tools/dump_mapid.py
"""

import json
import pathlib
import sys

import ee

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1]))

from services.gee_service import initialize_gee  # noqa: E402
from services.index_service import compute_all_indices  # noqa: E402
from utils.geo_utils import geojson_to_ee_geometry  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
OUT = REPO / "internal" / "gee" / "eeexpr" / "testdata"
POLY = REPO / "testdata" / "polygons"

NDVI_PALETTE = [
    '#ad0028', '#c5142a', '#e02d2c', '#ef4c3a', '#fe6c4a',
    '#ff8d5a', '#ffab69', '#ffc67d', '#ffe093', '#ffefab',
    '#fdfec2', '#eaf7ac', '#d5ef94', '#b9e383', '#9bd873',
    '#77ca6f', '#53bd6b', '#14aa60', '#009755', '#007e47', '#007e47',
]

_captured: dict = {}


def _install_interceptor() -> None:
    """Wrap ee.data's maps().create so we see the literal request body."""
    original = ee.data._get_cloud_projects

    def patched():
        projects = original()
        real_maps = projects.maps

        def maps_wrapper(*a, **kw):
            resource = real_maps(*a, **kw)
            real_create = resource.create

            def create_wrapper(**kwargs):
                _captured["parent"] = kwargs.get("parent")
                _captured["fields"] = kwargs.get("fields")
                _captured["body"] = kwargs.get("body")
                return real_create(**kwargs)

            resource.create = create_wrapper
            return resource

        projects.maps = maps_wrapper
        return projects

    ee.data._get_cloud_projects = patched


def main() -> int:
    if not initialize_gee():
        print("GEE initialisation failed.", file=sys.stderr)
        return 1

    _install_interceptor()

    geom = geojson_to_ee_geometry(json.loads((POLY / "nashik_vineyard.json").read_text()))
    collection = (
        ee.ImageCollection("COPERNICUS/S2_SR_HARMONIZED")
        .filterBounds(geom)
        .filterDate("2025-01-01", "2025-04-01")
        .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", 20))
        .map(lambda img: img.divide(10000))
    )
    indexed = compute_all_indices(collection.median())

    smooth = (
        indexed.select("NDVI")
        .clip(geom)
        .updateMask(indexed.select("NDVI").gte(0))
        .resample("bicubic")
        .reproject(crs="EPSG:4326", scale=10)
        .focal_mean(2, "circle", "pixels")
    )

    map_id = smooth.getMapId({"min": 0.0, "max": 1.0, "palette": NDVI_PALETTE})
    url_format = map_id["tile_fetcher"].url_format

    OUT.mkdir(parents=True, exist_ok=True)
    payload = {
        "request": {
            "parent": _captured.get("parent"),
            "fields": _captured.get("fields"),
            "body": _captured.get("body"),
        },
        "response": {
            "mapid": map_id.get("mapid"),
            "url_format": url_format,
        },
    }
    (OUT / "maps_request_response.json").write_text(
        json.dumps(payload, indent=2, sort_keys=True), encoding="utf-8"
    )

    body = _captured.get("body") or {}
    print("request body top-level keys:", sorted(body.keys()))
    for k in sorted(body.keys()):
        if k != "expression":
            print(f"  {k} = {json.dumps(body[k])[:400]}")
    print("\nmapid      :", map_id.get("mapid"))
    print("url_format :", url_format)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
