# How to Run

## Environment Requirements

- **Python**: 3.10+
- **Node.js**: 20+
- **PostgreSQL**: 16 with PostGIS 3.x extension enabled
- **Google Cloud Account**: GCP project with Earth Engine API enabled
- **Firebase Project**: Firebase console project with Phone Auth enabled
- **Ollama**: Local Ollama instance hosting `llama3.2` model

---

## 1. Backend Setup & Run

```powershell
# Navigate to backend directory
cd backend

# Create and activate Python virtual environment
python -m venv venv
# On Windows PowerShell:
.\venv\Scripts\Activate.ps1
# On Linux/macOS:
source venv/bin/activate

# Install backend dependencies
pip install -r requirements.txt -r chatbot\requirements.txt

# Authenticate Google Earth Engine (one-time setup)
earthengine authenticate

# Copy environment variables configuration
copy .env.example .env
# Set GEE_PROJECT_ID, DATABASE_URL, and JWT_SECRET_KEY in .env

# Run Flask REST API
python app.py
```
Backend will start on `http://127.0.0.1:5000`.

---

## 2. Frontend Setup & Run

```powershell
# Navigate to frontend directory
cd frontend

# Install npm packages
npm install

# Run Vite dev server
npm run dev
```
Frontend will start on `http://localhost:5173`.

---

## 3. Database Setup

### PostgreSQL + PostGIS:
```sql
psql -U postgres
CREATE DATABASE mindstrix;
\c mindstrix
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
\i schema.sql
```

---

## 4. Ollama Chatbot Service Setup

```bash
# Start Ollama service
ollama serve

# Pull llama3.2 model
ollama pull llama3.2
```

## Related Documents
- [17_CONFIGURATION.md](./architecture/17_CONFIGURATION.md)
- [11_DATABASE.md](./architecture/11_DATABASE.md)
