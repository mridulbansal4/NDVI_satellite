# 14 — Layer System & Color Palettes

## Multi-Layer Visualization

MindstriX supports multi-index visualization with custom continuous color palettes defined in `backend/app.py` and `frontend/src/colorUtils.js`.

## Active Layer Modes
1. **Optical Layer Mode (Sentinel-2)**:
   - **NDVI / EVI / SAVI / NDMI / NDWI / GNDVI**: Uses continuous EOS-style gradient (`NDVI_PALETTE`) transitioning from deep red (low vegetation) through yellow to vibrant green (high vegetation).
   - **CVI Composite**: Uses 3-tier health classification palette (`CVI_PALETTE`: `#ef4444`, `#f59e0b`, `#22c55e`).

2. **Radar Layer Mode (Sentinel-1 SAR)**:
   - **SMI / VV / VH (Moisture-Oriented)**: Sequential moisture gradient (`RADAR_MOISTURE_PALETTE`) from dry red/yellow to moist green/blue.
   - **RVI / VV-VH Ratio (Structural)**: Sequential blue gradient (`RADAR_SEQUENTIAL_PALETTE`).

## Mode Switcher (`LayerToggle.jsx`)
Applies state toggles switching between Optical composites and Sentinel-1 SAR moisture maps without clearing farm boundary state.

## Related Documents
- [06_VEGETATION_INDICES.md](./06_VEGETATION_INDICES.md)
- [13_MAP_RENDERING.md](./13_MAP_RENDERING.md)
