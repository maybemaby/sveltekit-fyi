-- +goose Up
CREATE INDEX IF NOT EXISTS idx_scans_eligible_domain
ON scans (domain)
WHERE (is_sk = 1 OR is_svelte = 1)
  AND (is_nsfw = 0 OR is_nsfw IS NULL);

ANALYZE;

-- +goose Down
DROP INDEX IF EXISTS idx_scans_eligible_domain;

ANALYZE;
