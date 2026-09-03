CREATE TABLE IF NOT EXISTS subscription_quota_reset_monitors (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    execution_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    interval_seconds INTEGER NOT NULL DEFAULT 600 CHECK (interval_seconds BETWEEN 60 AND 3600),
    drop_threshold_percent DOUBLE PRECISION NOT NULL DEFAULT 1 CHECK (drop_threshold_percent BETWEEN 1 AND 100),
    credit_policy VARCHAR(16) NOT NULL DEFAULT 'ignore' CHECK (credit_policy IN ('ignore', 'propagate')),
    reset_daily BOOLEAN NOT NULL DEFAULT FALSE,
    reset_weekly BOOLEAN NOT NULL DEFAULT TRUE,
    reset_monthly BOOLEAN NOT NULL DEFAULT FALSE,
    reset_five_hour BOOLEAN NOT NULL DEFAULT FALSE,
    account_ids BIGINT[] NOT NULL DEFAULT '{}',
    subscription_ids BIGINT[] NOT NULL DEFAULT '{}',
    last_checked_at TIMESTAMPTZ,
    next_check_at TIMESTAMPTZ,
    last_status VARCHAR(32) NOT NULL DEFAULT 'observing',
    last_error TEXT NOT NULL DEFAULT '',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS subscription_quota_reset_monitors_due_idx
    ON subscription_quota_reset_monitors (next_check_at)
    WHERE enabled = TRUE;

CREATE TABLE IF NOT EXISTS subscription_quota_reset_monitor_states (
    monitor_id BIGINT NOT NULL REFERENCES subscription_quota_reset_monitors(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL,
    previous_utilization_percent DOUBLE PRECISION,
    previous_reset_at TIMESTAMPTZ,
    previous_credit_hash TEXT NOT NULL DEFAULT '',
    previous_credit_count INTEGER NOT NULL DEFAULT 0,
    sampled_at TIMESTAMPTZ,
    candidate JSONB,
    last_error TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (monitor_id, account_id)
);

CREATE TABLE IF NOT EXISTS subscription_quota_reset_monitor_events (
    id UUID PRIMARY KEY,
    monitor_id BIGINT NOT NULL REFERENCES subscription_quota_reset_monitors(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL UNIQUE,
    classification VARCHAR(24) NOT NULL,
    status VARCHAR(24) NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    source_snapshot JSONB NOT NULL DEFAULT '[]',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS subscription_quota_reset_monitor_events_monitor_idx
    ON subscription_quota_reset_monitor_events (monitor_id, created_at DESC);

CREATE TABLE IF NOT EXISTS subscription_quota_reset_monitor_event_targets (
    event_id UUID NOT NULL REFERENCES subscription_quota_reset_monitor_events(id) ON DELETE CASCADE,
    subscription_id BIGINT NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    skip_reason TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    attempted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (event_id, subscription_id)
);

CREATE INDEX IF NOT EXISTS subscription_quota_reset_monitor_event_targets_pending_idx
    ON subscription_quota_reset_monitor_event_targets (event_id, status);
