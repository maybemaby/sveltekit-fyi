package internal

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var queryTracer = otel.Tracer("github.com/maybemaby/sveltekit-fyi/internal")

func errDeferLog(callback func() error, msg string) {
	err := callback()

	if err != nil {
		fmt.Println(msg)
	}
}

type AppStore struct {
	db *sql.DB

	statsMu        sync.RWMutex
	statsCache     CombinedStats
	statsCached    bool
	statsExpiresAt time.Time
	statsTTL       time.Duration
	now            func() time.Time
}

func NewAppStore(db *sql.DB) *AppStore {
	return &AppStore{
		db:       db,
		statsTTL: 24 * time.Hour,
		now:      time.Now,
	}
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

const getScanByDomain = `SELECT domain, scanned_at, is_sk, is_svelte, confidence, signals,
final_url, title, error, screenshot_path, og_image, redirected_to, is_nsfw
FROM scans WHERE domain = ?`

type Scan struct {
	Domain         string  `db:"domain"`
	ScannedAt      int     `db:"scanned_at"`
	IsSvelteKit    bool    `db:"is_sk"`
	IsSvelte       *bool   `db:"is_svelte"`
	Confidence     int     `db:"confidence"`
	Signals        string  `db:"signals"`
	FinalURL       *string `db:"final_url"`
	Title          *string `db:"title"`
	Error          *string `db:"error"`
	ScreenshotPath *string `db:"screenshot_path"`
	OGImage        *string `db:"og_image"`
	RedirectedTo   *string `db:"redirected_to"`
	IsNSFW         *bool   `db:"is_nsfw"`
}

func (s *AppStore) GetScanByDomain(ctx context.Context, domain string) (*Scan, error) {
	row := s.db.QueryRowContext(ctx, getScanByDomain, domain)

	var scan Scan

	err := row.Scan(
		&scan.Domain,
		&scan.ScannedAt,
		&scan.IsSvelteKit,
		&scan.IsSvelte,
		&scan.Confidence,
		&scan.Signals,
		&scan.FinalURL,
		&scan.Title,
		&scan.Error,
		&scan.ScreenshotPath,
		&scan.OGImage,
		&scan.RedirectedTo,
		&scan.IsNSFW,
	)

	if err != nil {
		return nil, err
	}

	return &scan, nil
}

const upsertScan = `INSERT INTO scans (domain, scanned_at, is_sk, is_svelte, confidence, signals, final_url, title, error, screenshot_path, og_image, redirected_to, is_nsfw)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		scan.IsSvelte,
		scan.Confidence,
		scan.Signals,
		scan.FinalURL,
		scan.Title,
		scan.Error,
		scan.ScreenshotPath,
		scan.OGImage,
		scan.RedirectedTo,
		scan.IsNSFW,
	)

	return err
}

func (s *AppStore) MarkScanNSFW(ctx context.Context, domain string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE scans SET is_nsfw = 1 WHERE domain = ?`, domain)

	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated for domain: %s", domain)
	}

	return nil
}

func (s *AppStore) AddScanError(ctx context.Context, domain string, errorMsg string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE scans SET error = ? WHERE domain = ?`, errorMsg, domain)
	return err
}

type DomainListing struct {
	Domain         string  `db:"domain" json:"domain"`
	FirstSeenAt    int     `db:"first_seen_at" json:"first_seen_at"`
	LastSeenAt     int     `db:"last_seen_at" json:"last_seen_at"`
	SeenCount      int     `db:"seen_count" json:"seen_count"`
	Signals        string  `db:"signals" json:"signals"`
	Title          *string `db:"title" json:"title"`
	OgImage        *string `db:"og_image" json:"og_image"`
	ScreenshotPath *string `db:"screenshot_path" json:"screenshot_path"`
	Total          int     `db:"total" json:"total"`
	IsSvelte       *bool   `db:"is_svelte" json:"is_svelte"`
	IsSvelteKit    bool    `db:"is_sk" json:"is_sk"`
}

const getTopDomains = `SELECT d.domain, d.first_seen_at, d.last_seen_at, d.seen_count,
  s.signals, s.title, s.og_image,
  (
    SELECT COUNT(*)
    FROM domains d2
    INNER JOIN scans s2 ON d2.domain = s2.domain
    WHERE (s2.is_sk = 1 OR s2.is_svelte = 1)
      AND (s2.is_nsfw = 0 OR s2.is_nsfw IS NULL)
  ) AS total,
  s.screenshot_path, s.is_svelte, s.is_sk
FROM domains d
INNER JOIN scans s ON d.domain = s.domain
WHERE (s.is_sk = 1 OR s.is_svelte = 1)
  AND (s.is_nsfw = 0 OR s.is_nsfw IS NULL)
ORDER BY %s
LIMIT ? OFFSET ?`

func (s *AppStore) GetTopDomains(ctx context.Context, order string, limit, offset int) ([]DomainListing, error) {
	ctx, span := queryTracer.Start(ctx, "db.query get_top_domains", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "sqlite"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.Int("db.limit", limit),
		attribute.Int("db.offset", offset),
	)

	ordering := map[string]string{
		"seen_at":    "d.first_seen_at DESC, d.domain ASC",
		"seen_count": "d.seen_count DESC, d.domain ASC",
	}

	orderBy, ok := ordering[order]

	if !ok {
		orderBy = "d.first_seen_at DESC, d.domain ASC"
	}

	query := fmt.Sprintf(getTopDomains, orderBy)

	rows, err := s.db.QueryContext(ctx, query, limit, offset)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "database query failed")
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
			&listing.ScreenshotPath,
			&listing.IsSvelte,
			&listing.IsSvelteKit,
		)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "database row scan failed")
			return nil, err
		}

		listings = append(listings, listing)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "database rows failed")
		return nil, err
	}

	return listings, nil
}

type ScanStats struct {
	ConfirmedSites  int `db:"confirmed_sites" json:"confirmedSites"`
	SvelteOnlySites int `db:"svelte_only_sites" json:"svelteOnlySites"`
	TotalScans      int `db:"total_scans" json:"totalScans"`
	TotalObserved   int `db:"total_observed" json:"totalObserved"`
}

const getStatsQuery = `SELECT (SELECT COUNT(*) FROM scans WHERE is_sk = true) AS confirmed_sites,
(SELECT COUNT(*) FROM scans WHERE is_sk = false AND is_svelte = true) AS svelte_only_sites,
(SELECT COUNT(domain) FROM scans) AS total_scans,
(SELECT COUNT(domain) FROM domains) AS total_observed
`

func (s *AppStore) getScanStats(ctx context.Context) (ScanStats, error) {
	row := s.db.QueryRowContext(ctx, getStatsQuery)

	var stats ScanStats

	err := row.Scan(
		&stats.ConfirmedSites,
		&stats.SvelteOnlySites,
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
	now := s.now()

	s.statsMu.RLock()
	if s.statsCached && now.Before(s.statsExpiresAt) {
		stats := s.statsCache
		s.statsMu.RUnlock()
		return stats, nil
	}
	s.statsMu.RUnlock()

	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	now = s.now()
	if s.statsCached && now.Before(s.statsExpiresAt) {
		return s.statsCache, nil
	}

	scanStats, err := s.getScanStats(ctx)

	if err != nil {
		return CombinedStats{}, err
	}

	signalCounts, err := s.getSignalCounts(ctx)

	if err != nil {
		return CombinedStats{}, err
	}

	stats := CombinedStats{
		Scans:   scanStats,
		Signals: signalCounts,
	}

	s.statsCache = stats
	s.statsCached = true
	s.statsExpiresAt = now.Add(s.statsTTL)

	return stats, nil
}

type SiteCountSnapshot struct {
	SnapshotAt     int `db:"snapshot_at" json:"snapshotAt"`
	KitCount int `db:"sk_count" json:"kitCount"`
	TotalScans     int `db:"total_scans" json:"totalScans"`
	TotalObserved  int `db:"total_observed" json:"totalObserved"`
	SvelteCount    int `db:"svelte_count" json:"svelteCount"`
}

const getSnapshotsQuery = `SELECT snapshot_at, sk_count, total_scans, total_observed, coalesce(svelte_count, 0) as svelte_count
FROM site_count
ORDER BY snapshot_at ASC
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
			&snapshot.KitCount,
			&snapshot.TotalScans,
			&snapshot.TotalObserved,
			&snapshot.SvelteCount,
		)

		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

type ScanToScreenshot struct {
	Domain         string  `db:"domain" json:"domain"`
	IsSvelteKit    bool    `db:"is_sk" json:"isSvelteKit"`
	ScreenshotPath *string `db:"screenshot_path" json:"screenshotPath"`
}

const getScansToScreenshotQuery = `SELECT domain, is_sk, screenshot_path from scans WHERE (is_sk = true OR is_svelte = true) AND error IS NULL AND screenshot_path IS NULL AND (og_image IS NULL OR og_image = '') LIMIT 1`

func (s *AppStore) GetScanToScreenshot(ctx context.Context) (ScanToScreenshot, error) {
	row := s.db.QueryRowContext(ctx, getScansToScreenshotQuery)

	var scan ScanToScreenshot

	err := row.Scan(
		&scan.Domain,
		&scan.IsSvelteKit,
		&scan.ScreenshotPath,
	)

	if err != nil {
		return ScanToScreenshot{}, err
	}

	return scan, nil
}

const updateScreenshotPathQuery = `UPDATE scans SET screenshot_path = ? WHERE domain = ?`

func (s *AppStore) UpdateScreenshotPath(ctx context.Context, domain string, screenshotPath string) error {
	_, err := s.db.ExecContext(ctx, updateScreenshotPathQuery, screenshotPath, domain)

	return err
}
