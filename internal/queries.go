package internal

import (
	"context"
	"database/sql"
)

type AppStore struct {
	db *sql.DB
}

func NewAppStore(db *sql.DB) *AppStore {
	return &AppStore{db: db}
}

const upsertDomainSeen = `
INSERT INTO domains (domain, first_seen_at, last_seen_at, seen_count)
VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
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
