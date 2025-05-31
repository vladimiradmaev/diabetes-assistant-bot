DO $$ 
BEGIN
    -- Only run the conversion if the tables exist and have the old column type
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'food_analyses' 
        AND column_name = 'confidence' 
        AND data_type = 'text'
    ) THEN
        -- First, add a temporary column for the conversion
        ALTER TABLE food_analyses ADD COLUMN confidence_new DOUBLE PRECISION;

        -- Convert string values to float8 using proper text comparison
        UPDATE food_analyses 
        SET confidence_new = CASE 
            WHEN confidence::text = 'high' THEN 0.9
            WHEN confidence::text = 'medium' THEN 0.6
            WHEN confidence::text = 'low' THEN 0.3
            ELSE 0.5
        END;

        -- Drop the old column and rename the new one
        ALTER TABLE food_analyses DROP COLUMN confidence;
        ALTER TABLE food_analyses RENAME COLUMN confidence_new TO confidence;

        -- Add constraints
        ALTER TABLE food_analyses 
            ALTER COLUMN confidence SET NOT NULL;

        ALTER TABLE food_analyses 
            ADD CONSTRAINT food_analyses_confidence_check 
            CHECK (confidence >= 0 AND confidence <= 1);
    END IF;

    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'food_analysis_corrections' 
        AND column_name = 'confidence' 
        AND data_type = 'text'
    ) THEN
        -- Do the same for food_analysis_corrections table
        ALTER TABLE food_analysis_corrections ADD COLUMN confidence_new DOUBLE PRECISION;

        UPDATE food_analysis_corrections 
        SET confidence_new = CASE 
            WHEN confidence::text = 'high' THEN 0.9
            WHEN confidence::text = 'medium' THEN 0.6
            WHEN confidence::text = 'low' THEN 0.3
            ELSE 0.5
        END;

        ALTER TABLE food_analysis_corrections DROP COLUMN confidence;
        ALTER TABLE food_analysis_corrections RENAME COLUMN confidence_new TO confidence;

        ALTER TABLE food_analysis_corrections 
            ALTER COLUMN confidence SET NOT NULL;

        ALTER TABLE food_analysis_corrections 
            ADD CONSTRAINT food_analysis_corrections_confidence_check 
            CHECK (confidence >= 0 AND confidence <= 1);
    END IF;
END $$; 