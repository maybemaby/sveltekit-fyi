package internal

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var svelteWindowRegex = regexp.MustCompile(`(?i)window\.__svelte\s*\?\?=\s*{}`)

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

var descriptionSelectors = []string{
	`meta[property="og:description"]`,
	`meta[name="og:description"]`,
	`meta[property="twitter:description"]`,
	`meta[name="twitter:description"]`,
	`meta[name="description"]`,
}

func probeDescription(doc *goquery.Document) string {
	for _, selector := range descriptionSelectors {

		if content, exists := doc.Find(selector).Attr("content"); exists {
			return content
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

func resolveRelativeUrl(baseUrl *url.URL, relativeUrl string) (*url.URL, error) {
	parsedRelativeUrl, err := url.Parse(relativeUrl)
	if err != nil {
		return nil, err
	}

	resolvedUrl := baseUrl.ResolveReference(parsedRelativeUrl)

	return resolvedUrl, nil
}

func detectSvelte(src io.Reader) bool {
	// Limit data read to 1.5MB to avoid reading too much data from large files
	limitReader := io.LimitReader(src, 1024*1024*1.5)
	bufReader := bufio.NewReader(limitReader)
	// Odds of a 22 byte string being split across two reads is low, 0.004% for 5 KB
	buffer := make([]byte, 1024*5)

	for {
		_, err := bufReader.Read(buffer)

		isSvelte := svelteWindowRegex.Match(buffer)

		if isSvelte {
			return true
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return false
		}
	}

	return false
}

func probeEntryUrls(doc *goquery.Document, pageUrl *url.URL) []string {
	var urls []string

	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")

		if exists && strings.HasSuffix(src, ".js") {
			resolvedUrl, err := resolveRelativeUrl(pageUrl, src)
			if err == nil && (resolvedUrl.Scheme == "http" || resolvedUrl.Scheme == "https") {
				urls = append(urls, resolvedUrl.String())
			}
		}
	})

	doc.Find(`link[rel="modulepreload"]`).Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")

		if exists && strings.HasSuffix(href, ".js") {
			resolvedUrl, err := resolveRelativeUrl(pageUrl, href)
			if err == nil && (resolvedUrl.Scheme == "http" || resolvedUrl.Scheme == "https") {
				urls = append(urls, resolvedUrl.String())
			}
		}
	})

	return urls
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
