CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGSERIAL PRIMARY KEY,
    service_name TEXT NOT NULL,
    price INTEGER NOT NULL CHECK (price >= 0),
    user_id UUID NOT NULL,
    start_date VARCHAR(7) NOT NULL,
    end_date VARCHAR(7),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_start_date_format CHECK (start_date ~ '^(0[1-9]|1[0-2])-\d{4}$'),
    CONSTRAINT check_end_date_format CHECK (end_date IS NULL OR end_date ~ '^(0[1-9]|1[0-2])-\d{4}$')
);
    
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_service_name ON subscriptions(service_name);