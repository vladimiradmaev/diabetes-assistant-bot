-- Add bread_units column to food_analyses table
ALTER TABLE food_analyses ADD COLUMN bread_units DOUBLE PRECISION;

-- Add bread_units column to food_analysis_corrections table
ALTER TABLE food_analysis_corrections ADD COLUMN bread_units DOUBLE PRECISION;

-- Update existing records to calculate bread units (carbs / 12)
UPDATE food_analyses SET bread_units = carbs / 12.0;
UPDATE food_analysis_corrections SET bread_units = corrected_carbs / 12.0;

-- Make the columns NOT NULL after updating existing records
ALTER TABLE food_analyses ALTER COLUMN bread_units SET NOT NULL;
ALTER TABLE food_analysis_corrections ALTER COLUMN bread_units SET NOT NULL; 