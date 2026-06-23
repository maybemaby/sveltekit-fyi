package internal

import (
	"context"
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
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}

// TODO: Avoid getting caught in a loop of failing to capture screenshots for invalid domains. We should mark them as failed and not try again.
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

	err = retry(3, time.Minute*2, func(retryAttempt int) error {

		if retryAttempt > 0 {
			s.logger.Warn("retrying screenshot capture", "domain", scan.Domain, "attempt", retryAttempt)
		}

		screenshot, err := s.renderer.Capture(ctx, scan.Domain)

		if err != nil {
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
		// Save the error to the database so we don't keep trying to capture a screenshot for this domain
		saveErr := s.appStore.AddScanError(ctx, scan.Domain, err.Error())

		if saveErr != nil {
			return saveErr
		}
	}

	return nil
}
