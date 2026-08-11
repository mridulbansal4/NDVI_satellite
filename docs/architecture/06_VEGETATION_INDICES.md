# 06 — Spectral & Radar Vegetation Indices

## Implemented Optical Indices (`services/index_service.py`)

All optical indices use scaled reflectance bands $[0.0, 1.0]$.

### 1. NDVI (Normalized Difference Vegetation Index)
$$\text{NDVI} = \frac{\text{NIR} - \text{RED}}{\text{NIR} + \text{RED}}$$
- **Code Function**: `_compute_ndvi(image)`
- **Target Bands**: B8 (NIR), B4 (RED)

### 2. EVI (Enhanced Vegetation Index)
$$\text{EVI} = 2.5 \times \frac{\text{NIR} - \text{RED}}{\text{NIR} + 6 \times \text{RED} - 7.5 \times \text{BLUE} + 1}$$
- **Code Function**: `_compute_evi(image)`
- **Target Bands**: B8 (NIR), B4 (RED), B2 (BLUE)

### 3. SAVI (Soil-Adjusted Vegetation Index)
$$\text{SAVI} = 1.5 \times \frac{\text{NIR} - \text{RED}}{\text{NIR} + \text{RED} + 0.5}$$
- **Code Function**: `_compute_savi(image)`
- **Target Bands**: B8 (NIR), B4 (RED)

### 4. NDMI (Normalized Difference Moisture Index)
$$\text{NDMI} = \frac{\text{NIR} - \text{SWIR}}{\text{NIR} + \text{SWIR}}$$
- **Code Function**: `_compute_ndmi(image)`
- **Target Bands**: B8 (NIR), B11 (SWIR)

### 5. NDWI (Normalized Difference Water Index)
$$\text{NDWI} = \frac{\text{GREEN} - \text{NIR}}{\text{GREEN} + \text{NIR}}$$
- **Code Function**: `_compute_ndwi(image)`
- **Target Bands**: B3 (GREEN), B8 (NIR)

### 6. GNDVI (Green Normalized Difference Vegetation Index)
$$\text{GNDVI} = \frac{\text{NIR} - \text{GREEN}}{\text{NIR} + \text{GREEN}}$$
- **Code Function**: `_compute_gndvi(image)`
- **Target Bands**: B8 (NIR), B3 (GREEN)

### 7. CVI (Composite Vegetation Index)
Weighted linear sum of all individual optical indices:
$$\text{CVI} = w_{\text{NDVI}} \cdot \text{NDVI} + w_{\text{EVI}} \cdot \text{EVI} + w_{\text{SAVI}} \cdot \text{SAVI} + w_{\text{NDMI}} \cdot \text{NDMI} + w_{\text{GNDVI}} \cdot \text{GNDVI}$$
- **Production Weights (`config.py`)**:
  - `NDVI`: 0.70
  - `EVI`: 0.10
  - `SAVI`: 0.05
  - `NDMI`: 0.10
  - `GNDVI`: 0.05

---

## Implemented Radar Indices (`services/radar_index_service.py`)

### 1. SMI (Soil Moisture Index)
$$\text{SMI} = \text{clamp}\left(\frac{\text{VV}_{\text{dB}} - (-20.0)}{-8.0 - (-20.0)}, 0.0, 1.0\right)$$

### 2. RVI (Radar Vegetation Index)
$$\text{RVI} = \frac{4 \cdot \text{VH}_{\text{linear}}}{\text{VV}_{\text{linear}} + \text{VH}_{\text{linear}}}$$

### 3. Ratio (VV/VH Ratio)
$$\text{Ratio} = \text{VV}_{\text{dB}} - \text{VH}_{\text{dB}}$$

## Related Documents
- [04_SENTINEL2_PIPELINE.md](./04_SENTINEL2_PIPELINE.md)
- [05_SENTINEL1_PIPELINE.md](./05_SENTINEL1_PIPELINE.md)
- [17_CONFIGURATION.md](./17_CONFIGURATION.md)
