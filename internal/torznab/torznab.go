// Package torznab queries Torznab/Newznab indexer feeds (as proxied by Prowlarr)
// and parses the results into release items.
package torznab

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Item is a single release returned by an indexer search.
type Item struct {
	Title       string
	Size        int64
	Seeders     int
	Leechers    int
	DownloadURL string // torrent file URL or .nzb URL
	MagnetURL   string
	InfoURL     string
	PubDate     string
}

// Client queries Torznab feeds.
type Client struct {
	http *http.Client
}

// New builds a client; a nil httpClient gets a sane default.
func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{http: httpClient}
}

// SearchURL builds a Torznab text-search URL from an indexer's settings.
func SearchURL(baseURL, apiPath, apiKey, query string, categories []int) string {
	base := strings.TrimRight(baseURL, "/")
	path := apiPath
	if path == "" {
		path = "/api"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	q := url.Values{}
	q.Set("t", "search")
	q.Set("q", query)
	if apiKey != "" {
		q.Set("apikey", apiKey)
	}
	if len(categories) > 0 {
		cats := make([]string, len(categories))
		for i, c := range categories {
			cats[i] = strconv.Itoa(c)
		}
		q.Set("cat", strings.Join(cats, ","))
	}
	return base + path + "?" + q.Encode()
}

// Search performs the query at the given fully-built Torznab URL.
func (c *Client) Search(ctx context.Context, searchURL string) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("torznab request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("torznab: status %d", resp.StatusCode)
	}
	return Parse(resp.Body)
}

type xmlRSS struct {
	Channel struct {
		Items []xmlItem `xml:"item"`
	} `xml:"channel"`
}

type xmlItem struct {
	Title     string `xml:"title"`
	GUID      string `xml:"guid"`
	Comments  string `xml:"comments"`
	PubDate   string `xml:"pubDate"`
	Size      int64  `xml:"size"`
	Link      string `xml:"link"`
	Enclosure struct {
		URL    string `xml:"url,attr"`
		Length int64  `xml:"length,attr"`
		Type   string `xml:"type,attr"`
	} `xml:"enclosure"`
	Attrs []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"attr"`
}

// Parse decodes a Torznab RSS document into items.
func Parse(r io.Reader) ([]Item, error) {
	var doc xmlRSS
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("torznab decode: %w", err)
	}
	items := make([]Item, 0, len(doc.Channel.Items))
	for _, x := range doc.Channel.Items {
		it := Item{
			Title:   strings.TrimSpace(x.Title),
			Size:    x.Size,
			InfoURL: x.Comments,
			PubDate: x.PubDate,
		}
		for _, a := range x.Attrs {
			switch strings.ToLower(a.Name) {
			case "seeders":
				it.Seeders = atoiSafe(a.Value)
			case "leechers":
				it.Leechers = atoiSafe(a.Value)
			case "peers":
				if it.Leechers == 0 {
					it.Leechers = atoiSafe(a.Value)
				}
			case "size":
				if it.Size == 0 {
					it.Size = atoi64Safe(a.Value)
				}
			}
		}
		// Resolve download/magnet links.
		switch {
		case strings.HasPrefix(x.Link, "magnet:"):
			it.MagnetURL = x.Link
		case x.Enclosure.URL != "":
			it.DownloadURL = x.Enclosure.URL
			if it.Size == 0 {
				it.Size = x.Enclosure.Length
			}
		default:
			it.DownloadURL = x.Link
		}
		items = append(items, it)
	}
	return items, nil
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atoi64Safe(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}
