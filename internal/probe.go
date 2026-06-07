package internal

import (
	"io"

	"github.com/PuerkitoBio/goquery"
)

var querySelectors = []string{
	"[data-sveltekit-preload-data]",
	"[data-sveltekit-keepalive]",
	"script[data-sveltekit-async-loader]",
	`[data-svelte-h^="svelte-"]`,
	"div#svelte-announcer",
}

func probeHTML(reader io.Reader) (bool, error) {
	for _, selector := range querySelectors {
		doc, err := goquery.NewDocumentFromReader(reader)

		if err != nil {
			return false, err
		}

		if doc.Find(selector).Length() > 0 {
			return true, nil
		}
	}

	return false, nil
}
