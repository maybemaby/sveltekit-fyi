package internal

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/maybemaby/sveltekit-fyi/internal/assert"
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

	selector, err := probeHTML(doc)

	if err != nil {
		t.Fatalf("failed to probe html for url %s: %v\n", url, err)
	}

	if selector == "" {
		t.Fatalf("expected url %s to be detected as SvelteKit, but it was not\n", url)
	}
}

func TestOgImage(t *testing.T) {
	t.Parallel()

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

	expected := "https://example.com/image.jpg"

	if ogImage != expected {
		t.Fatalf("expected og image to be %s, but got %s\n", expected, ogImage)
	}
}

func TestOgImageNotFound(t *testing.T) {
	t.Parallel()

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

	if ogImage != "" {
		t.Fatalf("expected og image to be empty string when not found, but got %s\n", ogImage)
	}
}

func TestOgImageFirst(t *testing.T) {
	t.Parallel()

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

	expected := "https://example.com/image.jpg"

	if ogImage != expected {
		t.Fatalf("expected og image to be %s, but got %s\n", expected, ogImage)
	}
}

func TestGetImageExtension(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "png image",
			url:  "https://podcasts.apple.com/assets/meta/apple-podcasts.png",
			want: ".png",
		},
		{
			name: "jpg image with query string",
			url:  "https://example.com/image.JPG?foo=bar",
			want: ".jpg",
		},
		{
			name: "no extension",
			url:  "https://example.com/assets/image",
			want: "",
		},
		{
			name: "invalid url",
			url:  "://not a valid url",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getImageExtension(tt.url)
			if got != tt.want {
				t.Fatalf("expected extension %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "description meta tag",
			html: `
				<html>
					<head>
						<meta name="description" content="This is a test description.">
					</head>
					<body></body>
				</html>
			`,
			want: "This is a test description.",
		},
		{
			name: "og:description meta tag",
			html: `
				<html>
					<head>
						<meta property="og:description" content="This is a test description.">
					</head>
					<body></body>
				</html>
			`,
			want: "This is a test description.",
		},
		{
			name: "twitter:description meta tag",
			html: `
				<html>
					<head>
						<meta name="twitter:description" content="This is a test description.">
					</head>
					<body></body>
				</html>
			`,
			want: "This is a test description.",
		},
		{
			name: "no description meta tags",
			html: `
				<html>
					<head></head>
					<body></body>
				</html>
			`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))

			if err != nil {
				t.Fatalf("failed to parse html: %v\n", err)
			}

			description := probeDescription(doc)

			if description != tt.want {
				t.Fatalf("expected description to be %s, but got %s\n", tt.want, description)
			}
		})
	}

}

func mustParseUrl(rawurl string) *url.URL {
	parsedUrl, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return parsedUrl
}

func TestResolveUrl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseUrl     string
		relativeUrl string
		want        string
	}{
		{
			name:        "absolute url",
			baseUrl:     "https://example.com",
			relativeUrl: "https://example.com/path/to/resource",
			want:        "https://example.com/path/to/resource",
		},
		{
			name:        "relative url",
			baseUrl:     "https://example.com",
			relativeUrl: "/path/to/resource",
			want:        "https://example.com/path/to/resource",
		},
		{
			name:        "relative url with query string",
			baseUrl:     "https://example.com",
			relativeUrl: "/path/to/resource?foo=bar",
			want:        "https://example.com/path/to/resource?foo=bar",
		},
		{
			name:        "parent directory",
			baseUrl:     "https://example.com/path/to",
			relativeUrl: "../resource",
			want:        "https://example.com/path/resource",
		},
		{
			name: "different domain",
			baseUrl:     "https://example.com",
			relativeUrl: "https://otherdomain.com/path/to/resource",
			want:        "https://otherdomain.com/path/to/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvedUrl, err := resolveRelativeUrl(mustParseUrl(tt.baseUrl), tt.relativeUrl)

			if err != nil {
				t.Fatalf("failed to resolve url: %v\n", err)
			}

			assert.Equal(t, resolvedUrl.String(), tt.want)
		})
	}
}

func TestDetectSvelte(t *testing.T) {

	tests := []struct {
		name string
		src  string
	}{
		{
			name: "svelte version",
			src:  `((window.__svelte ??= {}).v ??= new Set()).add(PUBLIC_VERSION);}`,
		},
		{
			name: "svelte uid",
			src:  `(window.__svelte ??= {}).uid ??= 1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSvelte(strings.NewReader(tt.src))
			assert.True(t, got)
		})
	}

}

func TestSortEntriesByScore(t *testing.T) {
	urls := []string{
		"https://static.example.com/assets/main.js",
		"https://static.example.com/assets/main.asd123.js",
		"https://static.example.com/assets/svelte.js",
		"https://static.example.com/somewhat/random.js",
	}

	sortedUrls := sortEntriesByScore(urls)

	expectedOrder := []string{
		"https://static.example.com/assets/main.asd123.js",
		"https://static.example.com/assets/main.js",
		"https://static.example.com/assets/svelte.js",
		"https://static.example.com/somewhat/random.js",
	}

	assert.Equal(t, expectedOrder, sortedUrls)
}
