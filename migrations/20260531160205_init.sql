-- +goose Up
CREATE TABLE IF NOT EXISTS domains (
  domain        TEXT PRIMARY KEY,
  first_seen_at INTEGER NOT NULL,
  last_seen_at  INTEGER NOT NULL,
  seen_count    INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS scans (
  domain          TEXT PRIMARY KEY,
  scanned_at      INTEGER NOT NULL,
  is_sk         INTEGER NOT NULL,
  confidence      INTEGER NOT NULL,
  signals         TEXT NOT NULL,
  final_url       TEXT,
  title           TEXT,
  screenshot_path TEXT,
  og_image        TEXT,
  redirected_to   TEXT,
  error           TEXT
);

CREATE TABLE IF NOT EXISTS notifications (
  domain     TEXT NOT NULL,
  channel    TEXT NOT NULL,
  posted_at  INTEGER NOT NULL,
  PRIMARY KEY (domain, channel)
);

CREATE TABLE IF NOT EXISTS reply_requests (
  post_uri      TEXT NOT NULL,
  post_cid      TEXT NOT NULL,
  root_uri      TEXT NOT NULL,
  root_cid      TEXT NOT NULL,
  author_did    TEXT NOT NULL,
  domain        TEXT NOT NULL,
  requested_at  INTEGER NOT NULL,
  replied_at    INTEGER,
  PRIMARY KEY (post_uri, domain)
);

CREATE INDEX IF NOT EXISTS idx_reply_requests_pending
  ON reply_requests (domain) WHERE replied_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS domains;
DROP TABLE IF EXISTS scans;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS reply_requests;
