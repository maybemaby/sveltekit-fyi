package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

var jetstreamUrl = "wss://jetstream2.us-west.bsky.network/subscribe?wantedCollections=app.bsky.feed.post"

type FacetFeature struct {
	Type string `json:"$type"`
	URI  string `json:"uri"`
	Did  string `json:"did"`
	Tag  string `json:"tag"`
}

type Facet struct {
	Features []FacetFeature `json:"features"`
	Index    struct {
		ByteStart int `json:"byteStart"`
		ByteEnd   int `json:"byteEnd"`
	} `json:"index"`
}

type Embed struct {
	Type     string `json:"$type"`
	External struct {
		URI         string `json:"uri"`
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"external"`
}

type JetstreamRecord struct {
	RecordType string   `json:"$type"`
	Text       string   `json:"text"`
	Langs      []string `json:"langs"`
	Facets     []Facet  `json:"facets"`
	Embed      *Embed   `json:"embed,omitempty"`
}

type JetstreamCommit struct {
	Operation  string          `json:"operation"`
	Collection string          `json:"collection"`
	RKey       string          `json:"rkey"`
	CID        string          `json:"cid"`
	Record     JetstreamRecord `json:"record"`
}

type JetstreamEvent struct {
	Did          string          `json:"did"`
	TimeUseconds int64           `json:"time_us"`
	Kind         string          `json:"kind"`
	Commit       JetstreamCommit `json:"commit"`
}

func extract_urls(event *JetstreamEvent) map[string]struct{} {
	urls := map[string]struct{}{}
	urlRegex := regexp.MustCompile("(?i)\\bhttps?://[^\\s<>\"'`)]+")

	if event.Commit.Record.Text == "" {
		return urls
	}

	if event.Commit.Record.Facets != nil {
		for _, facet := range event.Commit.Record.Facets {
			for _, feature := range facet.Features {
				if feature.Type == "app.bsky.richtext.facet#link" && feature.URI != "" {
					urls[feature.URI] = struct{}{}
				}
			}
		}
	}

	if event.Commit.Record.Embed != nil && event.Commit.Record.Embed.External.URI != "" {
		urls[event.Commit.Record.Embed.External.URI] = struct{}{}
	}

	if len(urls) == 0 && event.Commit.Record.Text != "" {
		matches := urlRegex.FindAllString(event.Commit.Record.Text, -1)
		for _, match := range matches {
			urls[match] = struct{}{}
		}
	}

	return urls
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func ProcessEvents(ctx context.Context, store *AppStore) error {
	backoff := 5 * time.Second
	httpClient := &http.Client{Timeout: 10 * time.Second}

	rateLimitHosts := map[string]time.Time{}
	hostsHit := map[string]struct{}{}

	c, _, err := websocket.Dial(ctx, jetstreamUrl, &websocket.DialOptions{})

	if err != nil {
		fmt.Printf("failed to connect to websocket: %v\n", err)

		return err
	}

	defer func() {
		err := c.CloseNow()
		if err != nil {
			fmt.Printf("failed to close websocket connection: %v\n", err)
		}
	}()

	for {

		select {
		case <-ctx.Done():
			err = c.Close(websocket.StatusNormalClosure, "")
			if err != nil {
				fmt.Printf("failed to close websocket connection after context done: %v\n", err)
			}
			return err
		default:
			event := &JetstreamEvent{}
			err := wsjson.Read(ctx, c, event)
			if err != nil {

				if errors.Is(err, context.Canceled) {
					fmt.Printf("context canceled, closing websocket connection\n")
					return err
				}

				fmt.Printf("failed to read websocket message: %v\n", err)

				var closeErr websocket.CloseError

				if errors.As(err, &closeErr) {
					fmt.Printf("websocket closed with status %d and reason: %s\n", closeErr.Code, closeErr.Reason)
					time.Sleep(backoff)
					backoff = minDuration(backoff*2, 1*time.Minute)
					// Reconnect to the websocket after a delay
					c.Close(websocket.StatusNormalClosure, "")
					c, _, err = websocket.Dial(ctx, jetstreamUrl, &websocket.DialOptions{})

					if err != nil {
						fmt.Printf("failed to reconnect to websocket: %v\n", err)
						continue
					}

					fmt.Println("Reconnected to websocket")
					backoff = 5 * time.Second
					continue
				}

				// Fallthrough and exit if it's some other kind of error besides a close error or context cancellation
				return err
			}

			urlsFound := extract_urls(event)

			for postUrl := range urlsFound {

				u, err := url.Parse(postUrl)

				if err != nil {
					fmt.Printf("failed to parse url %s: %v\n", postUrl, err)
					continue
				}

				host := u.Scheme + "://" + u.Hostname()

				fmt.Printf("Found URL: %s for event event: bsky.app %s\n\n", host, event.Commit.RKey)

				req, err := http.NewRequest(http.MethodGet, host, nil)

				if err != nil {
					fmt.Printf("failed to create request for url %s: %v\n", host, err)
					continue
				}

				err = store.AddDomainSeen(ctx, host)

				if err != nil {
					fmt.Printf("failed to add domain seen for domain %s: %v\n", host, err)
					return err
				}

				if rateLimitHosts[req.URL.Host].After(time.Now()) {
					fmt.Printf("skipping url %s due to recent rate limit\n", host)
					continue
				} else if _, hit := hostsHit[req.URL.Host]; hit {
					fmt.Printf("skipping url %s because we've already hit this host\n", host)
					continue
				}

				req.Header.Set("accept-language", "en")
				req.Header.Set("accept", "text/html,application/xhtml+xml")
				req.Header.Set("user-agent", "Mozilla/5.0 (compatible; SvelteKit-FYI/1.0; +https://brandma.dev)")

				resp, err := httpClient.Do(req)

				if err != nil {
					fmt.Printf("failed to fetch url %s: %v\n", host, err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					fmt.Printf("non-200 status code for url %s: %d\n", host, resp.StatusCode)

					if resp.StatusCode == http.StatusTooManyRequests {
						rateLimitHosts[req.URL.Host] = time.Now().Add(1 * time.Hour)
					}

					err = resp.Body.Close()

					if err != nil {
						fmt.Printf("failed to close response body for url %s: %v\n", host, err)
					}

					continue
				}

				doc, err := goquery.NewDocumentFromReader(resp.Body)

				if err != nil {
					fmt.Printf("failed to parse html for url %s: %v\n", host, err)
					err = resp.Body.Close()

					if err != nil {
						fmt.Printf("failed to close response body for url %s after parse failure: %v\n", host, err)
					}

					continue
				}

				isSvelteKit, err := probeHTML(doc)

				if err != nil {
					fmt.Printf("failed to probe html for url %s: %v\n", host, err)
					err = resp.Body.Close()

					if err != nil {
						fmt.Printf("failed to close response body for url %s after probe failure: %v\n", host, err)
					}

					continue
				}

				err = resp.Body.Close()

				if err != nil {
					fmt.Printf("failed to close response body for url %s after probe: %v\n", host, err)
				}

				if isSvelteKit {
					fmt.Printf("URL %s is likely a SvelteKit app\n", host)

					ogImage := probeOgImage(doc)

					if ogImage != "" {
						fmt.Printf("Found OG image for %s: %s\n", host, ogImage)
					}

				} else {
					fmt.Printf("URL %s does not appear to be a SvelteKit app\n\n", host)
				}

				hostsHit[req.URL.Host] = struct{}{}
			}
		}
	}
}
