package internal

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var USER_AGENT = "Mozilla/5.0 (compatible; SvelteKit-FYI/1.0; +https://sveltekit.fyi)"
var CHUNKS_LIMIT = 5
var svelteWindowRegex = regexp.MustCompile(`(?i)window\.__svelte\s*\?\?=\s*{}`)
var chunkScoring = []chunkScore{
	{regex: regexp.MustCompile(`(?i)(main|index|entry|index)[.-][a-z0-9]+\.m?js`), score: 40},
	{regex: regexp.MustCompile(`(?i)(main|index|entry|index)\.m?js`), score: 30},
	{regex: regexp.MustCompile(`(?i)\/assets[\/\.\-].*\.m?js`), score: 10},
}

var querySelectors = []string{
	"[data-sveltekit-preload-data]",
	"[data-sveltekit-keepalive]",
	"script[data-sveltekit-async-loader]",
	`[data-svelte-h^="svelte-"]`,
	"div#svelte-announcer",
}

type chunkScore struct {
	regex *regexp.Regexp
	score int
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

func resolveParentDomain(host string) string {
	parts := strings.Split(host, ".")

	if len(parts) < 2 {
		return host
	}

	return strings.Join(parts[len(parts)-2:], ".")
}

func DetectSvelte(src io.Reader) bool {
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

func sortEntriesByScore(entries []string) []string {
	type scoredEntry struct {
		url   string
		score int
	}

	var scoredEntries []scoredEntry

	for _, entry := range entries {
		score := 0

		for _, cs := range chunkScoring {
			if cs.regex.MatchString(entry) {
				score = max(score, cs.score)
			}
		}

		scoredEntries = append(scoredEntries, scoredEntry{url: entry, score: score})
	}

	slices.SortFunc(scoredEntries, func(a, b scoredEntry) int {
		return cmp.Compare(a.score, b.score)
	})

	slices.Reverse(scoredEntries)

	var sortedEntries []string

	for _, se := range scoredEntries {
		sortedEntries = append(sortedEntries, se.url)
	}

	return sortedEntries
}

func ProbeEntryUrls(doc *goquery.Document, pageUrl *url.URL) []string {
	var urls []string
	parentDomain := resolveParentDomain(pageUrl.Host)

	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")

		if exists && strings.HasSuffix(src, ".js") {
			resolvedUrl, err := resolveRelativeUrl(pageUrl, src)

			// Ignore scripts that are not on the same top level domain, too many third party scripts.
			// Care more about capturing sites intentionally using Svelte.
			if err == nil && (resolvedUrl.Scheme == "http" || resolvedUrl.Scheme == "https") && parentDomain == resolveParentDomain(resolvedUrl.Host) {
				urls = append(urls, resolvedUrl.String())
			}
		}
	})

	doc.Find(`link[rel="modulepreload"]`).Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")

		if exists && strings.HasSuffix(href, ".js") {
			resolvedUrl, err := resolveRelativeUrl(pageUrl, href)
			if err == nil && (resolvedUrl.Scheme == "http" || resolvedUrl.Scheme == "https") && parentDomain == resolveParentDomain(resolvedUrl.Host) {
				urls = append(urls, resolvedUrl.String())
			}
		}
	})

	sortedEntries := sortEntriesByScore(urls)
	topEntries := sortedEntries[:min(len(sortedEntries), CHUNKS_LIMIT)]

	return topEntries
}

func CollectChunks(ctx context.Context, c *http.Client, urls []string) [][]byte {
	var chunks [][]byte
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Go(func() {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

			if err != nil {
				// Silently skip
				return
			}

			req.Header.Set("user-agent", USER_AGENT)
			req.Header.Set("accept", "application/javascript, text/javascript, */*; q=0.01")

			resp, err := c.Do(req)

			if err != nil {
				// Silently skip
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				// Silently skip
				return
			}

			chunk, err := io.ReadAll(resp.Body)

			if err != nil {
				// Silently skip
				return
			}

			mu.Lock()
			chunks = append(chunks, chunk)
			mu.Unlock()
		})
	}

	wg.Wait()

	return chunks
}

func CheckSvelteForUrl(ctx context.Context, c *http.Client, doc *goquery.Document, pageUrl *url.URL) bool {
	urls := ProbeEntryUrls(doc, pageUrl)

	chunks := CollectChunks(ctx, c, urls)

	for _, chunk := range chunks {
		if DetectSvelte(bytes.NewReader(chunk)) {
			return true
		}
	}

	return false
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
