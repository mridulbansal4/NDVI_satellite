# 12 — Krishi Mitra AI Chatbot Architecture

## Overview
Krishi Mitra is an offline-capable agronomic AI assistant built as a dedicated Flask Blueprint in `backend/chatbot/`.

```
User Message + Live Farm Data (Frontend)
                  |
                  v
       POST /chatbot/chat (routes.py)
                  |
                  v
prompts/build_system_prompt(farm_data, heatmap_data)
  - Injects field name, polygon area, CVI mean,
    individual index values (NDVI, NDMI, etc.)
  - Formats strict agronomic advisor persona
                  |
                  v
         chain.py: ChatOllama Chain
  - Connects to local Ollama server (http://localhost:11434)
  - Model: OLLAMA_MODEL (default: llama3.2)
                  |
                  v
         memory.py: Session History
  - In-process memory store keyed by session_id
  - Capped at CHATBOT_MAX_HISTORY (default 10 turns)
                  |
                  v
         Response Streamed to Client
```

## System Prompt Context Grounding
The chatbot does not guess field state. On every conversation turn, `build_system_prompt()` dynamically formats live statistics:
- Current Vegetation Health (CVI, NDVI, EVI)
- Water & Drought Stress Indicators (NDMI, NDWI)
- Nitrogen / Chlorophyll Proxy (GNDVI)
- Confidence score and scene count

## Configuration Parameters (`backend/chatbot/config.py`)
- `OLLAMA_BASE_URL`: `http://localhost:11434`
- `OLLAMA_MODEL`: `llama3.2`
- `CHATBOT_MAX_HISTORY`: `10`

## Related Documents
- [07_API_ARCHITECTURE.md](./07_API_ARCHITECTURE.md)
- [17_CONFIGURATION.md](./17_CONFIGURATION.md)
