-- Add food_analyses columns that might be missing from the original migration
ALTER TABLE food_analyses ADD COLUMN IF NOT EXISTS bread_units DOUBLE PRECISION DEFAULT 0;
ALTER TABLE food_analyses ADD COLUMN IF NOT EXISTS insulin_ratio DOUBLE PRECISION DEFAULT 0;
ALTER TABLE food_analyses ADD COLUMN IF NOT EXISTS insulin_units DOUBLE PRECISION DEFAULT 0; 