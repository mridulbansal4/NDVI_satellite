# 01 — Introduction

## Platform Vision
The MindstriX Satellite Agronomy Intelligence Platform is engineered to deliver high-resolution satellite remote sensing analysis to precision agricultural workflows. By processing multi-spectral Sentinel-2 and radar-based Sentinel-1 satellite imagery in real-time, MindstriX transforms raw satellite spectral bands into actionable spatial heatmaps and agronomic recommendations.

## Core Capabilities
- **Composite Vegetation Index (CVI)**: Multi-index fusion blending NDVI, EVI, SAVI, NDMI, and GNDVI to prevent canopy saturation and soil background distortion.
- **Radar Soil Moisture (SAR)**: Sentinel-1 C-band backscatter analysis yielding Soil Moisture Index (SMI) and Radar Vegetation Index (RVI) unhindered by cloud cover.
- **Per-Cell Heatmap Tiling**: Dynamic polygon tiling generating 10m grid cells with Gaussian smoothing.
- **Krishi Mitra Grounded LLM Assistant**: Local AI agronomy bot utilizing grounded live farm stats for natural language agronomic Q&A.
- **Resumable Mobile-First Onboarding**: 9-step farmer onboarding flow synced with Cloud Firestore.

## Target Audience
- Precision Agronomists
- Farm Management Organizations & Cooperatives
- Agri-Fintech & Crop Insurance Assessment Teams

## Related Documents
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md)
- [02_SYSTEM_OVERVIEW.md](./02_SYSTEM_OVERVIEW.md)
