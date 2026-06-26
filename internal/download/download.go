// Package download sends grabbed releases to a download client (qBittorrent for
// torrents, SABnzbd for usenet).
package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Release is the minimal info needed to hand a grab to a download client.
type Release struct {
	Title       string
	DownloadURL string // .torrent or .nzb URL
	MagnetURL   string
	Protocol    string // torrent | usenet
}

func (r Release) link() string {
	if r.MagnetURL != "" {
		return r.MagnetURL
	}
	return r.DownloadURL
}

// Config is a resolved download-client configuration.
type Config struct {
	Implementation string
	Host           string
	Port           int
	UseSsl         bool
	URLBase        string
	Username       string
	Password       string
	APIKey         string
	Category       string
}

func (c Config) baseURL() string {
	scheme := "http"
	if c.UseSsl {
		scheme = "https"
	}
	u := fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
	if b := strings.Trim(c.URLBase, "/"); b != "" {
		u += "/" + b
	}
	return u
}

// Client sends releases to and tests a download client.
type Client interface {
	Add(ctx context.Context, rel Release) error
	TestConnection(ctx context.Context) error
}

// New builds a Client for the configured implementation.
func New(cfg Config, hc *http.Client) (Client, error) {
	switch strings.ToLower(cfg.Implementation) {
	case "qbittorrent":
		return newQbit(cfg, hc), nil
	case "sabnzbd":
		return &sab{cfg: cfg, http: ensureHTTP(hc)}, nil
	default:
		return nil, fmt.Errorf("download: unsupported client %q", cfg.Implementation)
	}
}

func ensureHTTP(hc *http.Client) *http.Client {
	if hc == nil {
		return &http.Client{Timeout: 30 * time.Second}
	}
	return hc
}

// --- qBittorrent (Web API v2) ---

type qbit struct {
	cfg  Config
	http *http.Client
}

func newQbit(cfg Config, hc *http.Client) *qbit {
	timeout := 30 * time.Second
	var transport http.RoundTripper
	if hc != nil {
		if hc.Timeout > 0 {
			timeout = hc.Timeout
		}
		transport = hc.Transport
	}
	jar, _ := cookiejar.New(nil)
	return &qbit{cfg: cfg, http: &http.Client{Timeout: timeout, Jar: jar, Transport: transport}}
}

func (q *qbit) login(ctx context.Context) error {
	form := url.Values{"username": {q.cfg.Username}, "password": {q.cfg.Password}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.cfg.baseURL()+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", q.cfg.baseURL())
	resp, err := q.http.Do(req)
	if err != nil {
		return fmt.Errorf("qbittorrent login: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// qBittorrent 5.x returns 204 (No Content) + the SID cookie on success;
	// older versions return 200 with the body "Ok." (and "Fails." for bad creds).
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	if resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "Ok." {
		return nil
	}
	return fmt.Errorf("qbittorrent login failed (status %d, body %q)", resp.StatusCode, strings.TrimSpace(string(body)))
}

func (q *qbit) Add(ctx context.Context, rel Release) error {
	if err := q.login(ctx); err != nil {
		return err
	}
	form := url.Values{"urls": {rel.link()}}
	if q.cfg.Category != "" {
		form.Set("category", q.cfg.Category)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.cfg.baseURL()+"/api/v2/torrents/add", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := q.http.Do(req)
	if err != nil {
		return fmt.Errorf("qbittorrent add: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qbittorrent add: status %d", resp.StatusCode)
	}
	return nil
}

func (q *qbit) TestConnection(ctx context.Context) error {
	if err := q.login(ctx); err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, q.cfg.baseURL()+"/api/v2/app/version", nil)
	resp, err := q.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qbittorrent version: status %d", resp.StatusCode)
	}
	return nil
}

// --- SABnzbd (HTTP API) ---

type sab struct {
	cfg  Config
	http *http.Client
}

func (s *sab) call(ctx context.Context, params url.Values) ([]byte, error) {
	params.Set("apikey", s.cfg.APIKey)
	params.Set("output", "json")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.baseURL()+"/api?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sabnzbd request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sabnzbd: status %d", resp.StatusCode)
	}
	return body, nil
}

func (s *sab) Add(ctx context.Context, rel Release) error {
	if rel.DownloadURL == "" {
		return fmt.Errorf("sabnzbd: release has no nzb URL")
	}
	params := url.Values{"mode": {"addurl"}, "name": {rel.DownloadURL}}
	if s.cfg.Category != "" {
		params.Set("cat", s.cfg.Category)
	}
	body, err := s.call(ctx, params)
	if err != nil {
		return err
	}
	var res struct {
		Status bool   `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err == nil && !res.Status {
		return fmt.Errorf("sabnzbd add failed: %s", res.Error)
	}
	return nil
}

func (s *sab) TestConnection(ctx context.Context) error {
	_, err := s.call(ctx, url.Values{"mode": {"version"}})
	return err
}
