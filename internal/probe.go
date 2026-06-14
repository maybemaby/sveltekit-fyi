package internal

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var querySelectors = []string{
	"[data-sveltekit-preload-data]",
	"[data-sveltekit-keepalive]",
	"script[data-sveltekit-async-loader]",
	`[data-svelte-h^="svelte-"]`,
	"div#svelte-announcer",
}

func probeHTML(doc *goquery.Document) (string, error) {
	for _, selector := range querySelectors {

		if doc.Find(selector).Length() > 0 {
			return selector, nil
		}
	}

	return "", nil
}

var titleSelectors = []string{
	`meta[property="og:title"]`,
	`meta[name="og:title"]`,
	`meta[property="twitter:title"]`,
	`meta[name="twitter:title"]`,
	"title",
}

func probeTitle(doc *goquery.Document) string {
	for _, selector := range titleSelectors {

		if content, exists := doc.Find(selector).Attr("content"); exists {
			return content
		}

		if selector == "title" {
			titleText := doc.Find("title").Text()
			if titleText != "" {
				return titleText
			}
		}
	}

	return ""
}

var ogImageSelectors = []string{
	`meta[property="og:image:secure_url"]`,
	`meta[name="og:image:secure_url"]`,
	`meta[property="og:image"]`,
	`meta[name="og:image"]`,
	`meta[property="twitter:image"]`,
	`meta[name="twitter:image"]`,
}

func probeOgImage(doc *goquery.Document) string {
	for _, selector := range ogImageSelectors {

		if content, exists := doc.Find(selector).Attr("content"); exists {
			return content
		}
	}

	return ""
}

func getImage(url string) (image []byte, err error) {
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer func() {
		closeErr := resp.Body.Close()

		if err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	// Limit to 5MB
	reader := io.LimitReader(resp.Body, 5242880)

	return io.ReadAll(reader)
}

func getImageExtension(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	ext := strings.ToLower(path.Ext(parsedURL.Path))
	if ext == "" || ext == "." {
		return ""
	}

	return ext
}
