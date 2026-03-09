-- Auto-applied on first container start via /docker-entrypoint-initdb.d

CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    labels JSONB,
    value DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_name ON metrics(name);
CREATE INDEX IF NOT EXISTS idx_timestamp ON metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_labels ON metrics USING GIN(labels);

CREATE TABLE IF NOT EXISTS metric_aggregates (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
