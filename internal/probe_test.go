package internal

import (
	"net/http"
	"testing"
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

	isSvelteKit, err := probeHTML(resp.Body)

	if err != nil {
		t.Fatalf("failed to probe html for url %s: %v\n", url, err)
	}

	if !isSvelteKit {
		t.Fatalf("expected url %s to be detected as SvelteKit, but it was not\n", url)
	}
}
