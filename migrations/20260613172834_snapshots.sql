-- +goose Up
CREATE TABLE IF NOT EXISTS site_count (
    sk_count INTEGER NOT NULL,
    total_scans INTEGER NOT NULL,
    total_observed INTEGER NOT NULL,
    snapshot_at   INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS site_count;
