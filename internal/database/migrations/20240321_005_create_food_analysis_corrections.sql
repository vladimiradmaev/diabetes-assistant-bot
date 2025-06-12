-- Add bread_units column to food_analysis_corrections if missing
ALTER TABLE food_analysis_corrections ADD COLUMN IF NOT EXISTS bread_units DOUBLE PRECISION DEFAULT 0; 