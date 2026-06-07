package internal

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestSveltekitOk(t *testing.T) {
	url := "https://svelte.dev/"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("accept-language", "en")
	req.Header.Set("accept", "text/html,application/xhtml+xml")

	if err != nil {
		t.Fatalf("failed to create request for url %s: %v\n", url, err)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatalf("failed to fetch url %s: %v\n", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("non-200 status code for url %s: %d\n", url, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		t.Fatalf("failed to parse html for url %s: %v\n", url, err)
	}

	isSvelteKit, err := probeHTML(doc)

	if err != nil {
		t.Fatalf("failed to probe html for url %s: %v\n", url, err)
	}

	if !isSvelteKit {
		t.Fatalf("expected url %s to be detected as SvelteKit, but it was not\n", url)
	}
}

func TestOgImage(t *testing.T) {

	html := `
		<html>
			<head>
				<meta property="og:image" content="https://example.com/image.jpg">
			</head>
			<body></body>
		</html>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {
		t.Fatalf("failed to parse html: %v\n", err)
	}

	ogImage := probeOgImage(doc)

	if err != nil {
		t.Fatalf("failed to probe og image: %v\n", err)
	}

	expected := "https://example.com/image.jpg"

	if ogImage != expected {
		t.Fatalf("expected og image to be %s, but got %s\n", expected, ogImage)
	}
}

func TestOgImageNotFound(t *testing.T) {

	html := `
		<html>
			<head></head>
			<body></body>
		</html>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {
		t.Fatalf("failed to parse html: %v\n", err)
	}

	ogImage := probeOgImage(doc)

	if err != nil {
		t.Fatalf("failed to probe og image: %v\n", err)
	}

	if ogImage != "" {
		t.Fatalf("expected og image to be empty string when not found, but got %s\n", ogImage)
	}
}

func TestOgImageFirst(t *testing.T) {

	html := `
		<html>
			<head>
				<meta name="og:image:secure_url" content="https://example.com/image.jpg">
				<meta property="og:image" content="https://example.com/image2.jpg">
			</head>
			<body></body>
		</html>
	`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {
		t.Fatalf("failed to parse html: %v\n", err)
	}

	ogImage := probeOgImage(doc)

	if err != nil {
		t.Fatalf("failed to probe og image: %v\n", err)
	}

	expected := "https://example.com/image.jpg"

	if ogImage != expected {
		t.Fatalf("expected og image to be %s, but got %s\n", expected, ogImage)
	}
}
