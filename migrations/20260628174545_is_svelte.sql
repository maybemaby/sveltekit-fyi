-- +goose Up
ALTER TABLE scans ADD COLUMN is_svelte INTEGER;
ALTER TABLE site_count ADD COLUMN svelte_count INTEGER;

-- +goose Down
ALTER TABLE scans DROP COLUMN is_svelte;
ALTER TABLE site_count DROP COLUMN svelte_count;
