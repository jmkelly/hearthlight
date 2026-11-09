package api

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
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

// syncBuffer is an io.Writer safe for concurrent SSE handlers.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// streamRecorder is a ResponseWriter that records deadlines and can simulate a
// client whose socket no longer accepts writes.
type streamRecorder struct {
	mu        sync.Mutex
	header    http.Header
	status    int
	body      bytes.Buffer
	deadlines []time.Time
	writeErr  error
}

func (w *streamRecorder) Header() http.Header {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *streamRecorder) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

func (w *streamRecorder) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status == 0 {
		w.status = code
	}
}

func (w *streamRecorder) Flush() {}

func (w *streamRecorder) SetWriteDeadline(deadline time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func (w *streamRecorder) recordedDeadlines() []time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]time.Time(nil), w.deadlines...)
}

func waitForClients(t *testing.T, app *App, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		app.mu.RLock()
		got := len(app.clients)
		app.mu.RUnlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("connected SSE clients = %d, want %d", got, want)
		}
		time.Sleep(time.Millisecond)
	}
}

func newTestApp(logs io.Writer) *App {
	return New(nil, Config{Logger: slog.New(slog.NewTextHandler(logs, nil))})
}

func newEventRequest(ctx context.Context) *http.Request {
	return httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/events", nil).WithContext(ctx)
}

func TestEventStreamRefusesClientsAtLimit(t *testing.T) {
	logs := &syncBuffer{}
	app := newTestApp(logs)
	handler := app.Handler(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < maxSSEClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.ServeHTTP(&streamRecorder{}, newEventRequest(ctx))
		}()
	}
	waitForClients(t, app, maxSSEClients)

	rejected := &streamRecorder{}
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rejected, newEventRequest(ctx))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("event stream at the client limit did not return")
	}
	if rejected.status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rejected.status, http.StatusServiceUnavailable)
	}
	if !strings.Contains(logs.String(), "sse client limit") {
		t.Fatalf("limit rejection was not logged: %q", logs.String())
	}

	cancel()
	wg.Wait()
	waitForClients(t, app, 0)
}

func TestEventStreamSetsWriteDeadlineForEveryWrite(t *testing.T) {
	app := newTestApp(io.Discard)
	handler := app.Handler(nil)
	w := &streamRecorder{}
	start := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(w, newEventRequest(ctx))
		close(done)
	}()
	waitForClients(t, app, 1)

	app.SetSnapshot(model.Snapshot{})
	waitForDeadlines(t, w, 2)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("event stream did not stop after cancellation")
	}

	now := time.Now()
	for _, deadline := range w.recordedDeadlines() {
		if deadline.IsZero() {
			t.Fatal("SSE write had no deadline")
		}
		if !deadline.After(start) {
			t.Fatalf("SSE write deadline %v is not in the future", deadline)
		}
		if deadline.After(now.Add(sseWriteTimeout)) {
			t.Fatalf("SSE write deadline %v is beyond %v", deadline, sseWriteTimeout)
		}
	}
	waitForClients(t, app, 0)
}

func waitForDeadlines(t *testing.T, w *streamRecorder, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for len(w.recordedDeadlines()) < want {
		if time.Now().After(deadline) {
			t.Fatalf("SSE deadlines recorded = %d, want at least %d", len(w.recordedDeadlines()), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestEventStreamDropsClientWhenWriteFails(t *testing.T) {
	app := newTestApp(io.Discard)
	handler := app.Handler(nil)
	w := &streamRecorder{writeErr: io.ErrClosedPipe}

	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/events", nil))
	waitForClients(t, app, 0)
}
