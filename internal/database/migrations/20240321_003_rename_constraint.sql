-- Rename the automatically created unique constraint to the name GORM expects
DO $$ 
DECLARE
    current_constraint_name VARCHAR(255);
BEGIN
    -- Find the current constraint name for telegram_id unique constraint
    SELECT constraint_name INTO current_constraint_name
    FROM information_schema.table_constraints 
    WHERE table_name = 'users' 
    AND constraint_type = 'UNIQUE'
    AND constraint_name LIKE '%telegram_id%'
    LIMIT 1;
    
    -- Rename it to what GORM expects if it exists and isn't already named correctly
    IF current_constraint_name IS NOT NULL AND current_constraint_name != 'uni_users_telegram_id' THEN
        EXECUTE format('ALTER TABLE users RENAME CONSTRAINT %I TO uni_users_telegram_id', current_constraint_name);
    END IF;
END $$; 