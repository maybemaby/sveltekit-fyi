package internal_test

import (
	"database/sql"
	"testing"

	"github.com/maybemaby/sveltekit-fyi/internal"
	"github.com/maybemaby/sveltekit-fyi/internal/assert"
	_ "modernc.org/sqlite"
)

func setupTestDB() *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")

	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
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
  error           TEXT,
  is_nsfw         INTEGER
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

CREATE TABLE IF NOT EXISTS site_count (
    sk_count INTEGER NOT NULL,
    total_scans INTEGER NOT NULL,
    total_observed INTEGER NOT NULL,
    snapshot_at   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reply_requests_pending
  ON reply_requests (domain) WHERE replied_at IS NULL;

  INSERT INTO domains (domain, first_seen_at, last_seen_at, seen_count) VALUES
		('https://example.com', strftime('%s', 'now'), strftime('%s', 'now'), 1),
		('https://nsfwsite.com', strftime('%s', 'now'), strftime('%s', 'now'), 1);

  INSERT INTO scans (domain, scanned_at, is_sk, confidence, signals, is_nsfw) VALUES
  ('https://example.com', strftime('%s', 'now'), 1, 100, '["signal1", "signal2"]', 0),
  ('https://nsfwsite.com', strftime('%s', 'now'), 0, 50, '["signal3"]', 1);
  `)

	if err != nil {
		panic(err)
	}

	return db
}

func TestAddDomainSeen(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	domain := "https://example.com"

	err := store.AddDomainSeen(t.Context(), domain)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAddDomainUpsert(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	domain := "https://example1.com"

	err := store.AddDomainSeen(t.Context(), domain)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = store.AddDomainSeen(t.Context(), domain)

	if err != nil {
		t.Fatalf("expected no error on upsert, got %v", err)
	}

	var seenCount int

	err = db.QueryRow("SELECT seen_count FROM domains WHERE domain = ?", domain).Scan(&seenCount)

	if err != nil {
		t.Fatalf("expected to find domain, got error: %v", err)
	}

	if seenCount != 2 {
		t.Fatalf("expected seen_count to be 2, got %d", seenCount)
	}
}

func TestGetScanByDomain(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	scan, err := store.GetScanByDomain(t.Context(), "https://example.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if scan.Domain != "https://example.com" {
		t.Errorf("expected domain to be https://example.com, got %s", scan.Domain)
	}
}

func TestAddScanError(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	err := store.AddScanError(t.Context(), "https://example.com", "Some error occurred")

	assert.Nil(t, err)
}

func TestMarkScanAsNSFW(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	err := store.MarkScanNSFW(t.Context(), "https://example.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMarkScanAsNSFWNotFound(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	err := store.MarkScanNSFW(t.Context(), "https://nonexistent.com")

	if err == nil {
		t.Fatalf("expected error for non-existent domain, got nil")
	}
}

func TestGetScanNotFound(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	_, err := store.GetScanByDomain(t.Context(), "https://nonexistent.com")

	if err == nil {
		t.Fatalf("expected error for non-existent domain, got nil")
	}

	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetTopDomains(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	domains, err := store.GetTopDomains(t.Context(), "seen_at", 10, 0)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(domains) != 1 {
		t.Fatalf("expected 1 domains, got %d", len(domains))
	}

	if domains[0].Domain != "https://example.com" {
		t.Errorf("expected first domain to be https://example.com, got %s", domains[0].Domain)
	}
}

func TestUpdateScreenshot(t *testing.T) {
	db := setupTestDB()

	store := internal.NewAppStore(db)

	err := store.UpdateScreenshotPath(t.Context(), "https://example.com", "/path/to/screenshot.png")

	assert.Nil(t, err)
}
