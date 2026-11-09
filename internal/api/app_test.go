package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func newTestHandler(t *testing.T, cfg Config) (http.Handler, chan struct{}) {
	t.Helper()
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	scans := make(chan struct{}, 1)
	return New(scans, cfg).Handler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("ok")},
	}), scans
}

func doRequest(handler http.Handler, method, target string, headers map[string]string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Host = "127.0.0.1:8090"
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestHandlerHostPolicy(t *testing.T) {
	handler, _ := newTestHandler(t, Config{AllowedHosts: []string{"hearthlight.lan"}})
	cases := []struct {
		name   string
		host   string
		status int
	}{
		{"ipv4 loopback", "127.0.0.1:8090", http.StatusOK},
		{"ipv6 loopback", "[::1]:8090", http.StatusOK},
		{"localhost", "localhost:8090", http.StatusOK},
		{"lan ip literal", "192.168.1.10:8090", http.StatusOK},
		{"configured hostname", "hearthlight.lan:8090", http.StatusOK},
		{"rebinding hostname", "rebind.attacker.example", http.StatusForbidden},
		{"empty host", "", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.invalid/api/state", nil)
			req.Host = tc.host
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
		})
	}
}

func TestHandlerTokenPolicy(t *testing.T) {
	handler, _ := newTestHandler(t, Config{Token: "s3cret"})
	cases := []struct {
		name    string
		target  string
		headers map[string]string
		status  int
	}{
		{"api without token", "/api/state", nil, http.StatusUnauthorized},
		{"api wrong token", "/api/state", map[string]string{"Authorization": "Bearer wrong"}, http.StatusUnauthorized},
		{"api bearer token", "/api/state", map[string]string{"Authorization": "Bearer s3cret"}, http.StatusOK},
		{"api custom header", "/api/state", map[string]string{"X-Hearthlight-Token": "s3cret"}, http.StatusOK},
		{"api query token", "/api/state?token=s3cret", nil, http.StatusOK},
		{"health without token", "/healthz", nil, http.StatusOK},
		{"page without token", "/", nil, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(handler, http.MethodGet, tc.target, tc.headers, nil)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
		})
	}
}

func TestHandlerTokenLoginSetsCookie(t *testing.T) {
	handler, _ := newTestHandler(t, Config{Token: "s3cret"})

	rec := doRequest(handler, http.MethodGet, "/?token=s3cret", nil, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/" {
		t.Fatalf("location = %q, want /", location)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %v, want one", cookies)
	}
	if cookie := cookies[0]; cookie.Name != tokenCookie || cookie.Value != "s3cret" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie = %+v", cookie)
	}

	rec = doRequest(handler, http.MethodGet, "/api/state", nil, cookies)
	if rec.Code != http.StatusOK {
		t.Fatalf("state with login cookie status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestScanRequiresSameOrigin(t *testing.T) {
	handler, scans := newTestHandler(t, Config{})
	cases := []struct {
		name    string
		headers map[string]string
		status  int
		queued  bool
	}{
		{"cross-origin", map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden, false},
		{"cross-site fetch", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden, false},
		{"origin host mismatch", map[string]string{"Origin": "http://127.0.0.1:9999"}, http.StatusForbidden, false},
		{"null origin", map[string]string{"Origin": "null"}, http.StatusForbidden, false},
		{"same-origin", map[string]string{"Origin": "http://127.0.0.1:8090", "Sec-Fetch-Site": "same-origin"}, http.StatusAccepted, true},
		{"non-browser", nil, http.StatusAccepted, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for len(scans) > 0 {
				<-scans
			}
			rec := doRequest(handler, http.MethodPost, "/api/scan", tc.headers, nil)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			queued := len(scans) > 0
			if queued != tc.queued {
				t.Fatalf("scan queued = %v, want %v", queued, tc.queued)
			}
		})
	}
}
