-- Ensure all required columns exist in food_analyses
DO $$ 
BEGIN
    -- Add columns if they don't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'food_analyses' AND column_name = 'bread_units') THEN
        ALTER TABLE food_analyses ADD COLUMN bread_units DOUBLE PRECISION DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'food_analyses' AND column_name = 'insulin_ratio') THEN
        ALTER TABLE food_analyses ADD COLUMN insulin_ratio DOUBLE PRECISION DEFAULT 0;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'food_analyses' AND column_name = 'insulin_units') THEN
        ALTER TABLE food_analyses ADD COLUMN insulin_units DOUBLE PRECISION DEFAULT 0;
    END IF;
    
    -- Add columns to food_analysis_corrections if they don't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'food_analysis_corrections' AND column_name = 'bread_units') THEN
        ALTER TABLE food_analysis_corrections ADD COLUMN bread_units DOUBLE PRECISION DEFAULT 0;
    END IF;
END $$; 