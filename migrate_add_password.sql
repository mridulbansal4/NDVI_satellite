-- Migration: Add password_hash column, make name nullable
ALTER TABLE farmers ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE farmers ALTER COLUMN name DROP NOT NULL;
