-- Migration: 001_initial_schema
-- Description: Create initial database schema for photo-tags application

-- Create images table for tracking image processing history
CREATE TABLE IF NOT EXISTS images (
    id SERIAL PRIMARY KEY,
    trace_id VARCHAR(255) NOT NULL UNIQUE,
    telegram_id BIGINT NOT NULL,
    telegram_username VARCHAR(255),
    filename VARCHAR(512) NOT NULL,
    original_path VARCHAR(1024),
    processed_path VARCHAR(1024),
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on telegram_id for faster user queries
CREATE INDEX IF NOT EXISTS idx_images_telegram_id ON images(telegram_id);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);

-- Create index on created_at for time-based queries
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at DESC);

-- Create index on trace_id for lookups
CREATE INDEX IF NOT EXISTS idx_images_trace_id ON images(trace_id);

-- Create processing_stats table for daily metrics
CREATE TABLE IF NOT EXISTS processing_stats (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    total_images INTEGER DEFAULT 0,
    successful_images INTEGER DEFAULT 0,
    failed_images INTEGER DEFAULT 0,
    pending_images INTEGER DEFAULT 0,
    total_users INTEGER DEFAULT 0,
    avg_processing_time_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on date for time-based queries
CREATE INDEX IF NOT EXISTS idx_processing_stats_date ON processing_stats(date DESC);

-- Create errors table for detailed error tracking and analysis
CREATE TABLE IF NOT EXISTS errors (
    id SERIAL PRIMARY KEY,
    trace_id VARCHAR(255),
    service VARCHAR(100) NOT NULL,
    error_type VARCHAR(100) NOT NULL,
    error_message TEXT NOT NULL,
    stack_trace TEXT,
    telegram_id BIGINT,
    filename VARCHAR(512),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on service for filtering
CREATE INDEX IF NOT EXISTS idx_errors_service ON errors(service);

-- Create index on error_type for analytics
CREATE INDEX IF NOT EXISTS idx_errors_type ON errors(error_type);

-- Create index on created_at for time-based queries
CREATE INDEX IF NOT EXISTS idx_errors_created_at ON errors(created_at DESC);

-- Create index on telegram_id for user error tracking
CREATE INDEX IF NOT EXISTS idx_errors_telegram_id ON errors(telegram_id);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at on images
CREATE TRIGGER update_images_updated_at BEFORE UPDATE ON images
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create trigger to automatically update updated_at on processing_stats
CREATE TRIGGER update_processing_stats_updated_at BEFORE UPDATE ON processing_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
