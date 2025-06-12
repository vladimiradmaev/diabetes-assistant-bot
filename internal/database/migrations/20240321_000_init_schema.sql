-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    telegram_id BIGINT NOT NULL UNIQUE,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    active_insulin_time INTEGER DEFAULT 0
);

-- Create food_analyses table
CREATE TABLE IF NOT EXISTS food_analyses (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id INTEGER REFERENCES users(id),
    image_url TEXT NOT NULL,
    weight DOUBLE PRECISION NOT NULL,
    carbs DOUBLE PRECISION NOT NULL,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    analysis_text TEXT NOT NULL,
    used_provider VARCHAR(10) NOT NULL CHECK (used_provider IN ('gemini', 'openai'))
);

-- Create food_analysis_corrections table
CREATE TABLE IF NOT EXISTS food_analysis_corrections (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id INTEGER REFERENCES users(id),
    original_carbs DOUBLE PRECISION NOT NULL,
    corrected_carbs DOUBLE PRECISION NOT NULL,
    original_weight DOUBLE PRECISION NOT NULL,
    corrected_weight DOUBLE PRECISION NOT NULL,
    image_url TEXT NOT NULL,
    analysis_text TEXT NOT NULL,
    used_provider VARCHAR(10) NOT NULL CHECK (used_provider IN ('gemini', 'openai')),
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1)
);

-- Create blood_sugar_records table
CREATE TABLE IF NOT EXISTS blood_sugar_records (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id INTEGER REFERENCES users(id),
    value DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create insulin_ratios table
CREATE TABLE IF NOT EXISTS insulin_ratios (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id INTEGER REFERENCES users(id),
    start_time VARCHAR(5) NOT NULL CHECK (start_time ~ '^([0-1][0-9]|2[0-3]):[0-5][0-9]$'),
    end_time VARCHAR(5) NOT NULL CHECK (end_time ~ '^([0-1][0-9]|2[0-3]):[0-5][0-9]$'),
    ratio DOUBLE PRECISION NOT NULL CHECK (ratio > 0)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_food_analyses_user_id ON food_analyses(user_id);
CREATE INDEX IF NOT EXISTS idx_food_analysis_corrections_user_id ON food_analysis_corrections(user_id);
CREATE INDEX IF NOT EXISTS idx_blood_sugar_records_user_id ON blood_sugar_records(user_id);
CREATE INDEX IF NOT EXISTS idx_blood_sugar_records_timestamp ON blood_sugar_records(timestamp);
CREATE INDEX IF NOT EXISTS idx_insulin_ratios_user_id ON insulin_ratios(user_id); 