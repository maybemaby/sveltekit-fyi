package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

const SNAPSHOT_INTERVAL = time.Hour * 24
const getLatestSnapshotTimeQuery = `SELECT snapshot_at FROM site_count ORDER BY snapshot_at DESC LIMIT 1`

func getLatestSnapshotTime(ctx context.Context, db *sql.DB) (time.Time, error) {
	var snapshotAt int64

	err := db.QueryRowContext(ctx, getLatestSnapshotTimeQuery).Scan(&snapshotAt)

	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(snapshotAt, 0), nil
}

const insertSnapshotQuery = `INSERT INTO site_count (snapshot_at, sk_count, total_scans, total_observed) VALUES (?, ?, ?, ?)`

func takeSnapshot(ctx context.Context, db *sql.DB) error {

	var scanStats ScanStats
	row := db.QueryRowContext(ctx, getStatsQuery)

	err := row.Scan(
		&scanStats.ConfirmedSites,
		&scanStats.TotalScans,
		&scanStats.TotalObserved,
	)

	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, insertSnapshotQuery, time.Now().Unix(), scanStats.ConfirmedSites, scanStats.TotalScans, scanStats.TotalObserved)

	if err != nil {
		return err
	}

	return nil
}

func RunSnapshots(ctx context.Context, db *sql.DB, logger *slog.Logger) error {

	snapLogger := logger.WithGroup("snapshots")

	timer := time.NewTicker(time.Hour)
	defer timer.Stop()

	// Create initial snapshot if none exist
	_, err := getLatestSnapshotTime(ctx, db)

	if err != nil {

		if err == sql.ErrNoRows {
			snapLogger.Info("no snapshots found, taking first snapshot")
			err = takeSnapshot(ctx, db)
			if err != nil {
				return fmt.Errorf("take snapshot: %w", err)
			}
		} else {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			snapshotAt, err := getLatestSnapshotTime(ctx, db)

			snapLogger.Info("latest snapshot", "snapshot_at", snapshotAt)

			if time.Since(snapshotAt) < SNAPSHOT_INTERVAL {
				snapLogger.Info("snapshot is recent, skipping")
				continue
			}

			err = takeSnapshot(ctx, db)

			if err != nil {
				return fmt.Errorf("take snapshot: %w", err)
			}
			snapLogger.Info("snapshot taken")
		}
	}
}
