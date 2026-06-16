package internal

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
)

// More fullproof would be to use litestream, but good enough for now
func BackupDB(ctx context.Context, db *sql.DB, s3 *S3Client) error {
	currentTime := time.Now().Unix()
	backupFileName := fmt.Sprintf("sveltekit-fyi-backup-%d.db", currentTime)
	_, err := db.ExecContext(ctx, "VACUUM INTO ?", backupFileName)

	if err != nil {
		return err
	}

	// Upload to S3
	backupFile, err := os.Open(backupFileName)
	if err != nil {
		return err
	}

	defer func() {
		errDeferLog(backupFile.Close, "failed to close backup file")
	}()

	key := fmt.Sprintf("backups/%s", backupFileName)

	return s3.uploadBackup(ctx, key, backupFile)
}
