package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"
)

var interval = time.Minute * 4

type ScreenshotService struct {
	renderer BrowserRenderer
	appStore *AppStore
	s3Client *S3Client
	logger   *slog.Logger
}

func NewScreenshotService(renderer BrowserRenderer, appStore *AppStore, s3Client *S3Client) *ScreenshotService {
	return &ScreenshotService{
		renderer: renderer,
		appStore: appStore,
		s3Client: s3Client,
		logger:   slog.New(slog.DiscardHandler),
	}
}

func (s *ScreenshotService) SetLogger(logger *slog.Logger) {
	s.logger = logger.WithGroup("screenshot_service")
}

func (s *ScreenshotService) Run(ctx context.Context) error {

	timer := time.NewTicker(interval)
	defer timer.Stop()

	// First run immediately, then run on the interval
	err := s.runLoop(ctx)

	if err != nil {
		return err
	}

	for {
		select {
		case <-timer.C:
			err := s.runLoop(ctx)

			if err != nil {

				// May run into a situation where there are no scans to screenshot, just wait for the next interval
				// Also, if we hit a rate limit, we should just wait for the next interval to try again
				if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrRateLimit) {
					continue
				}

				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}

func (s *ScreenshotService) runLoop(ctx context.Context) error {

	scan, err := s.appStore.GetScanToScreenshot(ctx)

	if err != nil {
		return err
	}

	parsedUrl, err := url.Parse(scan.Domain)

	if err != nil {
		// All saved domains should be valid URLs, but if not, we should exit
		return err
	}

	err = retry(ctx, 3, time.Minute*2, func(retryAttempt int) error {

		if retryAttempt > 0 {
			s.logger.Warn("retrying screenshot capture", "domain", scan.Domain, "attempt", retryAttempt)
		}

		screenshot, err := s.renderer.Capture(ctx, scan.Domain)

		if err != nil {

			if errors.Is(err, ErrRateLimit) {
				return errors.Join(err, ErrExitEarly)
			}

			return err
		}

		key := fmt.Sprintf("screenshots/%s.png", parsedUrl.Hostname())

		imgBytes, err := io.ReadAll(screenshot)

		if err != nil {
			return err
		}

		err = s.s3Client.UploadImage(ctx, key, imgBytes)

		if err != nil {
			return err
		}

		err = s.appStore.UpdateScreenshotPath(ctx, scan.Domain, key)

		if err != nil {
			return err
		}

		s.logger.Debug("screenshot captured and uploaded", "domain", scan.Domain, "s3_key", key)

		return nil
	})

	if err != nil {
		s.logger.Error("failed to capture screenshot", "domain", scan.Domain, "error", err)

		return handleRetryError(err, scan.Domain, s.appStore)
	}

	return nil
}

func handleRetryError(err error, domain string, appStore *AppStore) error {

	if errors.Is(err, ErrRateLimit) {
		return err
	}

	saveErr := appStore.AddScanError(context.Background(), domain, err.Error())

	if saveErr != nil {
		return saveErr
	}

	return nil
}
