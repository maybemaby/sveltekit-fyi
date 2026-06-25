-- +goose Up
UPDATE scans
SET error = NULL
WHERE error = 'unexpected status code 429';

-- +goose Down

