# Project Context

## Overview
MindstriX is a precision-agriculture satellite intelligence platform designed to bridge complex Earth observation data with actionable farm-level insights. 

Agricultural monitoring traditionally suffers from a key bottleneck: standard single-index optical satellite products (such as raw NDVI) fail under common field conditions like dense canopy saturation or atmospheric interference. MindstriX addresses this by fusing multi-spectral optical data (Sentinel-2) and synthetic aperture radar data (Sentinel-1) using Google Earth Engine (GEE).

## Agriculture & Remote Sensing Context

### 1. Optical Remote Sensing (Sentinel-2)
The European Space Agency's Sentinel-2 mission provides high-resolution multi-spectral optical imagery (10m–20m spatial resolution). 
- **Bands Utilized**: Blue (B2), Green (B3), Red (B4), Near-Infrared / NIR (B8), Short-Wave Infrared / SWIR (B11), and Scene Classification Layer (SCL).
- **Primary Challenges**: Cloud cover, haze, and soil reflectance in early growth stages.

### 2. Synthetic Aperture Radar / SAR (Sentinel-1)
Sentinel-1 provides C-band Synthetic Aperture Radar imagery.
- **Polarizations Utilized**: VV (Vertical Transmit / Vertical Receive) and VH (Vertical Transmit / Horizontal Receive) backscatter.
- **Primary Advantages**: Unaffected by solar illumination or cloud cover, providing all-weather soil moisture and canopy structure insights.

### 3. Grounded AI Intelligence
Field stats generated from satellite composites serve as real-time context for **Krishi Mitra**, a localized LLM (Ollama + LangChain), enabling conversational agronomic advice directly related to the user's specific field parameters.

## Related Documents
- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [01_INTRODUCTION.md](./01_INTRODUCTION.md)
- [06_VEGETATION_INDICES.md](./architecture/06_VEGETATION_INDICES.md)
