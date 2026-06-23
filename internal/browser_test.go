package internal_test

import (
	"io"
	"os"
	"testing"

	"github.com/maybemaby/sveltekit-fyi/internal"
	"github.com/maybemaby/sveltekit-fyi/internal/assert"
)

func TestCloudflareRenderer(t *testing.T) {

	apiKey := os.Getenv("CLOUDFLARE_API_KEY")
	accountId := os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	if apiKey == "" || accountId == "" {
		t.Skip("Skipping test because CLOUDFLARE_API_KEY or CLOUDFLARE_ACCOUNT_ID is not set")
	}

	renderer, err := internal.NewCloudflareRenderer(accountId, apiKey)

	assert.Nil(t, err)

	img, err := renderer.Capture(t.Context(), "https://svelte.dev/")

	assert.Nil(t, err)

	// Verify there's some bytes

	data := make([]byte, 100)

	read, err := io.ReadAtLeast(img, data, 100)

	assert.Nil(t, err)
	assert.Equal(t, read, 100)
}
