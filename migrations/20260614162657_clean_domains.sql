-- +goose Up
DELETE FROM domains where domain LIKE '%www.%';
DELETE FROM scans where domain LIKE '%www.%';

-- +goose Down
