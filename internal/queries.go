package internal

import (
	"context"
	"database/sql"
	"fmt"
)

func errDeferLog(callback func() error, msg string) {
	err := callback()

	if err != nil {
		fmt.Println(msg)
	}
}

type AppStore struct {
	db *sql.DB
}

func NewAppStore(db *sql.DB) *AppStore {
	return &AppStore{db: db}
}

const upsertDomainSeen = `
INSERT INTO domains (domain, first_seen_at, last_seen_at, seen_count)
VALUES (?, strftime('%s', 'now'), strftime('%s', 'now'), 1)
ON CONFLICT (domain) DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at,
seen_count = seen_count + 1
`

func (s *AppStore) AddDomainSeen(ctx context.Context, domain string) error {

	_, err := s.db.ExecContext(ctx, upsertDomainSeen, domain)

	return err
}

const getScanByDomain = `SELECT * FROM scans WHERE domain = ?`

type Scan struct {
	Domain         string  `db:"domain"`
	ScannedAt      int     `db:"scanned_at"`
	IsSvelteKit    bool    `db:"is_sk"`
	Confidence     int     `db:"confidence"`
	Signals        string  `db:"signals"`
	FinalURL       *string `db:"final_url"`
	Title          *string `db:"title"`
	Error          *string `db:"error"`
	ScreenshotPath *string `db:"screenshot_path"`
	OGImage        *string `db:"og_image"`
	RedirectedTo   *string `db:"redirected_to"`
}

func (s *AppStore) GetScanByDomain(ctx context.Context, domain string) (*Scan, error) {
	row := s.db.QueryRowContext(ctx, getScanByDomain, domain)

	var scan Scan

	err := row.Scan(
		&scan.Domain,
		&scan.ScannedAt,
		&scan.IsSvelteKit,
		&scan.Confidence,
		&scan.Signals,
		&scan.FinalURL,
		&scan.Title,
		&scan.Error,
		&scan.ScreenshotPath,
		&scan.OGImage,
		&scan.RedirectedTo,
	)

	if err != nil {
		return nil, err
	}

	return &scan, nil
}

const upsertScan = `INSERT INTO scans (domain, scanned_at, is_sk, confidence, signals, final_url, title, error, screenshot_path, og_image, redirected_to)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (domain) DO UPDATE SET
scanned_at = EXCLUDED.scanned_at,
is_sk = EXCLUDED.is_sk,
confidence = EXCLUDED.confidence,
signals = EXCLUDED.signals,
final_url = EXCLUDED.final_url,
title = EXCLUDED.title,
error = EXCLUDED.error,
screenshot_path = EXCLUDED.screenshot_path,
og_image = EXCLUDED.og_image,
redirected_to = EXCLUDED.redirected_to
`

func (s *AppStore) SaveScan(ctx context.Context, scan *Scan) error {
	_, err := s.db.ExecContext(ctx, upsertScan,
		scan.Domain,
		scan.ScannedAt,
		scan.IsSvelteKit,
		scan.Confidence,
		scan.Signals,
		scan.FinalURL,
		scan.Title,
		scan.Error,
		scan.ScreenshotPath,
		scan.OGImage,
		scan.RedirectedTo,
	)

	return err
}

type DomainListing struct {
	Domain      string  `db:"domain" json:"domain"`
	FirstSeenAt int     `db:"first_seen_at" json:"first_seen_at"`
	LastSeenAt  int     `db:"last_seen_at" json:"last_seen_at"`
	SeenCount   int     `db:"seen_count" json:"seen_count"`
	Signals     string  `db:"signals" json:"signals"`
	Title       *string `db:"title" json:"title"`
	OgImage     *string `db:"og_image" json:"og_image"`
	Total       int     `db:"total" json:"total"`
}

const getTopDomains = `WITH top_domains AS (
  SELECT d.domain, d.first_seen_at, d.last_seen_at, d.seen_count, s.signals, s.title, s.og_image
  FROM domains d
  INNER JOIN scans s ON d.domain = s.domain
  WHERE s.is_sk = true
), counted_domains AS (
  SELECT *, COUNT(*) OVER () AS total
  FROM top_domains
)
SELECT domain, first_seen_at, last_seen_at, seen_count, signals, title, og_image, total
FROM counted_domains
ORDER BY first_seen_at DESC
LIMIT ? OFFSET ?`

func (s *AppStore) GetTopDomains(ctx context.Context, limit, offset int) ([]DomainListing, error) {
	rows, err := s.db.QueryContext(ctx, getTopDomains, limit, offset)

	if err != nil {
		return nil, err
	}

	defer errDeferLog(rows.Close, "failed to close rows")

	listings := make([]DomainListing, 0)

	for rows.Next() {
		var listing DomainListing

		err := rows.Scan(
			&listing.Domain,
			&listing.FirstSeenAt,
			&listing.LastSeenAt,
			&listing.SeenCount,
			&listing.Signals,
			&listing.Title,
			&listing.OgImage,
			&listing.Total,
		)

		if err != nil {
			return nil, err
		}

		listings = append(listings, listing)
	}

	return listings, nil
}

type ScanStats struct {
	ConfirmedSites int `db:"confirmed_sites" json:"confirmedSites"`
	TotalScans     int `db:"total_scans" json:"totalScans"`
	TotalObserved  int `db:"total_observed" json:"totalObserved"`
}

const getStatsQuery = `SELECT (SELECT COUNT(*) FROM scans WHERE is_sk = true) AS confirmed_sites,
(SELECT COUNT(domain) FROM scans) AS total_scans,
(SELECT COUNT(domain) FROM domains) AS total_observed
`

func (s *AppStore) getScanStats(ctx context.Context) (ScanStats, error) {
	row := s.db.QueryRowContext(ctx, getStatsQuery)

	var stats ScanStats

	err := row.Scan(
		&stats.ConfirmedSites,
		&stats.TotalScans,
		&stats.TotalObserved,
	)

	if err != nil {
		return ScanStats{}, err
	}

	return stats, nil
}

const getSignalCountsQuery = `SELECT signals, COUNT(*) AS count
FROM scans
WHERE is_sk = true
GROUP BY signals
ORDER BY count DESC
`

type SignalCount struct {
	Signals string `db:"signals" json:"signals"`
	Count   int    `db:"count" json:"count"`
}

func (s *AppStore) getSignalCounts(ctx context.Context) ([]SignalCount, error) {
	rows, err := s.db.QueryContext(ctx, getSignalCountsQuery)

	if err != nil {
		return nil, err
	}

	defer errDeferLog(rows.Close, "failed to close rows")

	counts := make([]SignalCount, 0)

	for rows.Next() {
		var count SignalCount

		err := rows.Scan(
			&count.Signals,
			&count.Count,
		)

		if err != nil {
			return nil, err
		}

		counts = append(counts, count)
	}

	return counts, nil
}

type CombinedStats struct {
	Scans   ScanStats     `json:"scans"`
	Signals []SignalCount `json:"signals"`
}

func (s *AppStore) GetStats(ctx context.Context) (CombinedStats, error) {
	scanStats, err := s.getScanStats(ctx)

	if err != nil {
		return CombinedStats{}, err
	}

	signalCounts, err := s.getSignalCounts(ctx)

	if err != nil {
		return CombinedStats{}, err
	}

	return CombinedStats{
		Scans:   scanStats,
		Signals: signalCounts,
	}, nil
}

type SiteCountSnapshot struct {
	SnapshotAt     int `db:"snapshot_at" json:"snapshotAt"`
	ConfirmedSites int `db:"sk_count" json:"confirmedSites"`
	TotalScans     int `db:"total_scans" json:"totalScans"`
	TotalObserved  int `db:"total_observed" json:"totalObserved"`
}

const getSnapshotsQuery = `SELECT snapshot_at, sk_count, total_scans, total_observed
FROM site_count
ORDER BY snapshot_at DESC
LIMIT 365
`

func (s *AppStore) GetSnapshots(ctx context.Context) ([]SiteCountSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, getSnapshotsQuery)

	if err != nil {
		return nil, err
	}

	defer errDeferLog(rows.Close, "failed to close rows")

	snapshots := make([]SiteCountSnapshot, 0)

	for rows.Next() {
		var snapshot SiteCountSnapshot

		err := rows.Scan(
			&snapshot.SnapshotAt,
			&snapshot.ConfirmedSites,
			&snapshot.TotalScans,
			&snapshot.TotalObserved,
		)

		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}
