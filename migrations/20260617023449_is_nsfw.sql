-- +goose Up
ALTER TABLE scans ADD COLUMN is_nsfw INTEGER;

-- +goose Down
ALTER TABLE scans DROP COLUMN is_nsfw;
