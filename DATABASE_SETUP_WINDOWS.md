# Database Setup — Windows (Beginner Guide)
# All commands are copy-paste ready for PowerShell / Command Prompt

## 1. Install PostgreSQL 16 + PostGIS

Download URL (Windows x86-64 installer):
  https://www.enterprisedb.com/downloads/postgres-postgresql-downloads
  → PostgreSQL 16 → Windows x86-64 → Download

Installer steps:
  1. Run the .exe as Administrator
  2. Select all components (PostgreSQL Server, pgAdmin 4, Stack Builder, Command Line Tools)
  3. Set a password for the 'postgres' user — REMEMBER THIS
  4. Port: 5432 (default — leave as-is)
  5. Locale: Default locale
  6. Click Install
  7. After install: Stack Builder launches automatically
     → Select PostgreSQL 16 on port 5432
     → Categories → Spatial Extensions → PostGIS 3.x for PostgreSQL 16
     → Download, Install, click Yes to create the spatial database

Mac one-liner:
  brew install postgresql@16 postgis

Linux (Ubuntu/Debian) one-liner:
  sudo apt install postgresql-16 postgresql-16-postgis-3


## 2. Create Database + Enable Extension

Open PowerShell or Command Prompt:

  psql -U postgres

Inside psql shell:

  CREATE DATABASE agri_platform;
  \c agri_platform
  CREATE EXTENSION postgis;
  SELECT PostGIS_Version();

Expected output from last command:
  postgis_version
  ----------------
  3.4 USE_GEOS=1 USE_PROJ=1 USE_STATS=1


## 3. Run Schema File

  psql -U postgres -d agri_platform -f C:\MY\MYPROJECTS\NDVI_satellite-main\schema.sql

Verify all tables created:

  psql -U postgres -d agri_platform
  \dt

Expected output (8 tables):
  farmers, farmer_locations, farms, crops, irrigation, soil_info, consents, vi_reports


## 4. .env File (Exact Format)

Create file: backend\.env (copy from backend\.env.example and fill in):

  DATABASE_URL=postgresql://postgres:yourpassword@localhost:5432/agri_platform
  FIREBASE_PROJECT_ID=mindstrix-3c2ae
  JWT_SECRET_KEY=supersecretkey_change_in_production
  FLASK_PORT=5000
  FLASK_ENV=development
  GEE_PROJECT_ID=mindstrix


## 5. Test Queries

-- Verify the seed farmer was inserted
  psql -U postgres -d agri_platform

  SELECT f.name, fl.district, fr.farm_name
  FROM farmers f
  JOIN farmer_locations fl ON fl.farmer_id = f.id
  JOIN farms fr ON fr.farmer_id = f.id
  WHERE f.mobile_number = '9876543210';

Expected output:
  name          | district | farm_name
  Ramesh Patil  | Nashik   | Ramesh's Vineyard

-- Check latest VI report
  SELECT farm_id, cvi_mean, ndvi, confidence_score, period_start, period_end
  FROM vi_reports ORDER BY created_at DESC LIMIT 1;


## 6. Common Errors + Exact Fixes

### ERROR: Port 5432 already in use
  # Stop the conflicting service (PowerShell as Admin):
  netstat -ano | findstr :5432
  # Find the PID in last column, then:
  taskkill /PID <PID> /F
  # Or restart PostgreSQL service:
  net stop postgresql-x64-16
  net start postgresql-x64-16

### ERROR: password authentication failed for user "postgres"
  # Reset password in psql:
  psql -U postgres
  ALTER USER postgres WITH PASSWORD 'newpassword';
  # Update DATABASE_URL in .env with the new password

### ERROR: type "geometry" does not exist (PostGIS not enabled)
  psql -U postgres -d agri_platform
  CREATE EXTENSION IF NOT EXISTS postgis;
  SELECT PostGIS_Version();
  # If this errors: PostGIS was not installed via Stack Builder
  # Re-run Stack Builder and install PostGIS 3.x for PostgreSQL 16

### ERROR: psql is not recognized as internal or external command
  # Add PostgreSQL bin directory to PATH:
  # Control Panel → System → Advanced → Environment Variables
  # Edit PATH → Add:
  C:\Program Files\PostgreSQL\16\bin
  # Then open a NEW PowerShell window and retry

### ERROR: could not connect to server: Connection refused
  # PostgreSQL service is not running. Start it:
  net start postgresql-x64-16
  # Or via Services app (services.msc) → PostgreSQL 16 → Start
