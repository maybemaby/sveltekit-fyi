package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

var rescanInterval = 30 * 24 * time.Hour
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

type JetStreamProcessor struct {
	store          *AppStore
	hostRateLimits map[string]time.Time
	hostHitsSet    map[string]struct{}
	logger         *slog.Logger
}

func NewJetStreamProcessor(store *AppStore) *JetStreamProcessor {
	return &JetStreamProcessor{
		store:          store,
		hostRateLimits: make(map[string]time.Time),
		hostHitsSet:    make(map[string]struct{}),
		logger:         slog.New(slog.DiscardHandler),
	}
}

func (p *JetStreamProcessor) SetLogger(logger *slog.Logger) {
	p.logger = logger.WithGroup("jetstream")
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

func createHostRequest(host string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, host, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create request for url %s: %w", host, err)
	}

	req.Header.Set("accept-language", "en")
	req.Header.Set("accept", "text/html,application/xhtml+xml")
	req.Header.Set("user-agent", "Mozilla/5.0 (compatible; SvelteKit-FYI/1.0; +https://brandma.dev)")

	return req, nil
}

func trySaveImage(url string, hostname string) error {
	img, err := getImage(url)

	if err != nil {
		return err
	}

	extension := getImageExtension(url)

	if extension == "" {
		return fmt.Errorf("could not determine image extension for url %s", url)
	}

	savePath := fmt.Sprintf("og/%s%s", hostname, extension)
	f, err := os.Create(savePath)

	if err != nil {
		return err
	}

	_, err = f.Write(img)

	if err != nil {
		return err
	}

	return f.Close()
}

func (p *JetStreamProcessor) ProcessEvents(ctx context.Context, store *AppStore) error {
	backoff := 5 * time.Second
	httpClient := &http.Client{Timeout: 10 * time.Second}
	retries := 0

	c, _, err := websocket.Dial(ctx, jetstreamUrl, &websocket.DialOptions{})

	if err != nil {
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
					return err
				}

				p.logger.Error("failed to read websocket message", "error", err.Error())

				var closeErr *websocket.CloseError

				if errors.As(err, &closeErr) {
					p.logger.Warn("websocket connection closed, attempting to reconnect", "code", closeErr.Code, "reason", closeErr.Reason)
				}

				time.Sleep(backoff)
				backoff = minDuration(backoff*2, 1*time.Minute)
				retries++
				// Reconnect to the websocket after a delay
				c.Close(websocket.StatusNormalClosure, "")
				c, _, err = websocket.Dial(ctx, jetstreamUrl, &websocket.DialOptions{})

				if err != nil {
					p.logger.Error("failed to reconnect to websocket", "error", err)

					if retries >= 5 {
						p.logger.Error("max retries reached, giving up on reconnecting to websocket")
						return err
					}

					continue
				}

				p.logger.Info("successfully reconnected to websocket")
				backoff = 5 * time.Second
				retries = 0
				continue
			}

			urlsFound := extract_urls(event)

			for postUrl := range urlsFound {

				u, err := url.Parse(postUrl)

				if err != nil {
					p.logger.Error("failed to parse url", "url", postUrl, "error", err)
					continue
				}

				if u.Scheme != "http" && u.Scheme != "https" {
					continue
				}

				host := u.Scheme + "://" + u.Hostname()

				err = store.AddDomainSeen(ctx, host)

				if err != nil {
					return err
				}

				p.logger.Info("found url in event", "url", host, "event", fmt.Sprintf("bsky.app %s", event.Commit.RKey))

				req, err := createHostRequest(host)

				if err != nil {
					p.logger.Error("failed to create request for url", "url", host, "error", err)
					continue
				}

				if p.hostRateLimits[req.URL.Host].After(time.Now()) {
					p.logger.Warn("skiping url due to recent rate limit", "host", host)
					continue
				} else if _, hit := p.hostHitsSet[req.URL.Host]; hit {
					p.logger.Warn("skipping url because we've already hit this host", "host", host)
					continue
				}

				resp, err := httpClient.Do(req)

				if err != nil {
					p.logger.Error("failed to fetch url", "url", host, "error", err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					fmt.Printf("non-200 status code for url %s: %d\n", host, resp.StatusCode)

					if resp.StatusCode == http.StatusTooManyRequests {
						p.hostRateLimits[req.URL.Host] = time.Now().Add(1 * time.Hour)
					}

					err = resp.Body.Close()

					if err != nil {
						p.logger.Debug("failed to close response body", "host", host, "error", err)
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
					p.logger.Info("URL appears to be a SvelteKit app", "host", host)

					ogImage := probeOgImage(doc)

					if ogImage != "" {
						fmt.Printf("Found OG image for %s: %s\n", host, ogImage)

						err = trySaveImage(ogImage, req.URL.Hostname())

						if err != nil {
							fmt.Printf("failed to save og image for url %s: %v\n", host, err)
						}
					}
				} else {
					p.logger.Debug("host does not appear to be a sveltekit app", "host", host)
				}

				p.hostHitsSet[req.URL.Host] = struct{}{}
			}
		}
	}
}
