package download

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func cfgFor(t *testing.T, impl, server string) Config {
	t.Helper()
	u, err := url.Parse(server)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	return Config{Implementation: impl, Host: host, Port: port}
}

func TestQbittorrentAdd(t *testing.T) {
	var gotURLs, gotCategory string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/add":
			_ = r.ParseForm()
			gotURLs = r.FormValue("urls")
			gotCategory = r.FormValue("category")
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/app/version":
			_, _ = w.Write([]byte("v4.6.0"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := cfgFor(t, "QBittorrent", srv.URL)
	cfg.Username, cfg.Password, cfg.Category = "admin", "pw", "movies"
	c, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := c.Add(context.Background(), Release{MagnetURL: "magnet:?xt=urn:btih:abc"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if gotURLs != "magnet:?xt=urn:btih:abc" || gotCategory != "movies" {
		t.Fatalf("qbit got urls=%q category=%q", gotURLs, gotCategory)
	}
	if err := c.TestConnection(context.Background()); err != nil {
		t.Fatalf("test connection: %v", err)
	}
}

// qBittorrent 5.x returns 204 (No Content) + cookie on login, not 200 "Ok.".
func TestQbittorrentLogin204(t *testing.T) {
	var added bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			http.SetCookie(w, &http.Cookie{Name: "QBT_SID_8080", Value: "abc"})
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/torrents/add":
			added = true
			_, _ = w.Write([]byte("Ok."))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	cfg := cfgFor(t, "QBittorrent", srv.URL)
	cfg.Username, cfg.Password = "admin", "pw"
	c, _ := New(cfg, nil)
	if err := c.Add(context.Background(), Release{MagnetURL: "magnet:?xt=urn:btih:abc"}); err != nil {
		t.Fatalf("add with 204 login: %v", err)
	}
	if !added {
		t.Fatal("torrent was not added after 204 login")
	}
}

func TestSabnzbdAdd(t *testing.T) {
	var gotName, gotMode, gotCat string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotMode = q.Get("mode")
		if gotMode == "addurl" {
			gotName = q.Get("name")
			gotCat = q.Get("cat")
		}
		_, _ = w.Write([]byte(`{"status":true,"nzo_ids":["SABnzbd_nzo_x"]}`))
	}))
	defer srv.Close()

	cfg := cfgFor(t, "Sabnzbd", srv.URL)
	cfg.APIKey, cfg.Category = "key", "movies"
	c, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := c.Add(context.Background(), Release{DownloadURL: "http://indexer/nzb/1"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if gotName != "http://indexer/nzb/1" || gotCat != "movies" {
		t.Fatalf("sab got name=%q cat=%q", gotName, gotCat)
	}
}

func TestSabnzbdAddFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":false,"error":"API Key Incorrect"}`))
	}))
	defer srv.Close()
	cfg := cfgFor(t, "Sabnzbd", srv.URL)
	c, _ := New(cfg, nil)
	if err := c.Add(context.Background(), Release{DownloadURL: "http://x/1"}); err == nil {
		t.Fatal("expected error on status:false")
	}
}

func TestUnsupportedClient(t *testing.T) {
	if _, err := New(Config{Implementation: "Transmission"}, nil); err == nil {
		t.Fatal("expected error for unsupported client")
	}
}
