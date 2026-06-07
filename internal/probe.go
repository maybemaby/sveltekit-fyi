package internal

import (
	"github.com/PuerkitoBio/goquery"
)

var querySelectors = []string{
	"[data-sveltekit-preload-data]",
	"[data-sveltekit-keepalive]",
	"script[data-sveltekit-async-loader]",
	`[data-svelte-h^="svelte-"]`,
	"div#svelte-announcer",
}

func probeHTML(doc *goquery.Document) (bool, error) {
	for _, selector := range querySelectors {

		if doc.Find(selector).Length() > 0 {
			return true, nil
		}
	}

	return false, nil
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
