package internal

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupStatsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE domains (
			domain        TEXT PRIMARY KEY,
			first_seen_at INTEGER NOT NULL,
			last_seen_at  INTEGER NOT NULL,
			seen_count    INTEGER NOT NULL DEFAULT 1
		);

		CREATE TABLE scans (
			domain          TEXT PRIMARY KEY,
			scanned_at      INTEGER NOT NULL,
			is_sk           INTEGER NOT NULL,
			is_svelte       INTEGER,
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

		INSERT INTO domains (domain, first_seen_at, last_seen_at, seen_count) VALUES
			('https://example.com', 1, 1, 1),
			('https://svelte.dev', 1, 1, 1);

		INSERT INTO scans (domain, scanned_at, is_sk, is_svelte, confidence, signals, is_nsfw) VALUES
			('https://example.com', 1, 1, NULL, 100, 'signal-a', 0),
			('https://svelte.dev', 1, 0, 1, 100, 'signal-b', 0);
	`)
	if err != nil {
		t.Fatalf("seed sqlite db: %v", err)
	}

	return db
}

func TestGetStatsUsesCachedValueWithinTTL(t *testing.T) {
	db := setupStatsTestDB(t)
	store := NewAppStore(db)

	now := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	stats, err := store.GetStats(t.Context())
	if err != nil {
		t.Fatalf("first GetStats failed: %v", err)
	}

	if stats.Scans.TotalObserved != 2 {
		t.Fatalf("expected 2 observed domains, got %d", stats.Scans.TotalObserved)
	}

	_, err = db.Exec(`
		INSERT INTO domains (domain, first_seen_at, last_seen_at, seen_count) VALUES ('https://new-site.com', 1, 1, 1);
		INSERT INTO scans (domain, scanned_at, is_sk, is_svelte, confidence, signals, is_nsfw) VALUES ('https://new-site.com', 1, 1, NULL, 100, 'signal-c', 0);
	`)
	if err != nil {
		t.Fatalf("insert new rows: %v", err)
	}

	cachedStats, err := store.GetStats(t.Context())
	if err != nil {
		t.Fatalf("second GetStats failed: %v", err)
	}

	if cachedStats.Scans.TotalObserved != 2 {
		t.Fatalf("expected cached observed count to remain 2, got %d", cachedStats.Scans.TotalObserved)
	}
	if cachedStats.Scans.TotalScans != 2 {
		t.Fatalf("expected cached scan count to remain 2, got %d", cachedStats.Scans.TotalScans)
	}
}

func TestGetStatsRefreshesAfterTTL(t *testing.T) {
	db := setupStatsTestDB(t)
	store := NewAppStore(db)

	now := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	store.statsTTL = time.Hour

	_, err := store.GetStats(t.Context())
	if err != nil {
		t.Fatalf("first GetStats failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO domains (domain, first_seen_at, last_seen_at, seen_count) VALUES ('https://new-site.com', 1, 1, 1);
		INSERT INTO scans (domain, scanned_at, is_sk, is_svelte, confidence, signals, is_nsfw) VALUES ('https://new-site.com', 1, 1, NULL, 100, 'signal-c', 0);
	`)
	if err != nil {
		t.Fatalf("insert new rows: %v", err)
	}

	now = now.Add(2 * time.Hour)

	refreshedStats, err := store.GetStats(t.Context())
	if err != nil {
		t.Fatalf("refreshed GetStats failed: %v", err)
	}

	if refreshedStats.Scans.TotalObserved != 3 {
		t.Fatalf("expected refreshed observed count to be 3, got %d", refreshedStats.Scans.TotalObserved)
	}
	if refreshedStats.Scans.TotalScans != 3 {
		t.Fatalf("expected refreshed scan count to be 3, got %d", refreshedStats.Scans.TotalScans)
	}
}
