package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
	"github.com/jmkelly/hearthlight/internal/store"
	"github.com/jmkelly/hearthlight/web"
)

func newTestHandler(t *testing.T, cfg Config) (http.Handler, chan struct{}) {
	t.Helper()
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	scans := make(chan struct{}, 1)
	return New(scans, cfg).Handler(web.Files), scans
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
		{"themes without token", "/api/themes", nil, http.StatusUnauthorized},
		{"themes bearer token", "/api/themes", map[string]string{"Authorization": "Bearer s3cret"}, http.StatusOK},
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

func TestThemesEndpoint(t *testing.T) {
	handler, _ := newTestHandler(t, Config{})
	rec := doRequest(handler, http.MethodGet, "/api/themes", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", got)
	}
	want, err := web.Files.ReadFile("themes.json")
	if err != nil {
		t.Fatalf("read embedded themes.json: %v", err)
	}
	if !bytes.Equal(rec.Body.Bytes(), want) {
		t.Fatal("response body does not match the embedded theme catalog")
	}
	var catalog struct {
		SchemaVersion int               `json:"schemaVersion"`
		Families      []json.RawMessage `json:"families"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catalog); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if catalog.SchemaVersion != 1 || len(catalog.Families) != 2 {
		t.Fatalf("schemaVersion = %d, families = %d, want 1 and 2", catalog.SchemaVersion, len(catalog.Families))
	}
}

func TestThemeAssetsAndIndexAreServed(t *testing.T) {
	handler, _ := newTestHandler(t, Config{})
	for _, target := range []string{"/", "/themes.json", "/themes.js"} {
		rec := doRequest(handler, http.MethodGet, target, nil, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("%s status = %d, want %d", target, rec.Code, http.StatusOK)
		}
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

func layoutDevice(ip, mac string) model.Device {
	return model.Device{IP: ip, MAC: mac}
}

func newLayoutTestHandler(t *testing.T, devices ...model.Device) (http.Handler, *App, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := store.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	if len(devices) > 0 {
		if err := st.Apply(devices, now, nil); err != nil {
			t.Fatal(err)
		}
	}
	app := New(nil, Config{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Layout: st,
	})
	app.SetSnapshot(st.Snapshot(store.SnapshotOptions{Now: now, Interval: time.Minute}))
	return app.Handler(web.Files), app, path
}

func doPositionRequest(handler http.Handler, target, contentType, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Host = "127.0.0.1:8090"
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func deviceFromStore(t *testing.T, path, id string) model.Device {
	t.Helper()
	st, err := store.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	snap := st.Snapshot(store.SnapshotOptions{Now: time.Now(), Interval: time.Minute})
	for _, d := range snap.Devices {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("device %s missing from reloaded store", id)
	return model.Device{}
}

func deviceFromApp(app *App, id string) model.Device {
	for _, d := range app.Snapshot().Devices {
		if d.ID == id {
			return d
		}
	}
	return model.Device{}
}

func TestDevicePositionAcceptsJSONAndForm(t *testing.T) {
	const id = "mac:aa:bb:cc:dd:ee:01"
	cases := []struct {
		name        string
		contentType string
		body        string
		x, y        float64
	}{
		{"json", "application/json", `{"x":12.5,"y":-7.25}`, 12.5, -7.25},
		{"json without content type", "", `{"x":3,"y":4}`, 3, 4},
		{"form", "application/x-www-form-urlencoded", "x=5.5&y=-6", 5.5, -6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, app, path := newLayoutTestHandler(t, layoutDevice("192.0.2.20", "aa:bb:cc:dd:ee:01"))
			rec := doPositionRequest(handler, "/api/devices/"+id+"/position", tc.contentType, tc.body, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body)
			}

			stored := deviceFromStore(t, path, id)
			if !stored.Pinned || stored.X != tc.x || stored.Y != tc.y {
				t.Fatalf("stored position = (%v, %v) pinned=%v, want (%v, %v) pinned", stored.X, stored.Y, stored.Pinned, tc.x, tc.y)
			}
			published := deviceFromApp(app, id)
			if !published.Pinned || published.X != tc.x || published.Y != tc.y {
				t.Fatalf("published position = (%v, %v) pinned=%v, want (%v, %v) pinned", published.X, published.Y, published.Pinned, tc.x, tc.y)
			}
		})
	}
}

func TestDevicePositionUnknownDeviceReturns404(t *testing.T) {
	handler, app, path := newLayoutTestHandler(t, layoutDevice("192.0.2.20", "aa:bb:cc:dd:ee:01"))
	rec := doPositionRequest(handler, "/api/devices/mac:aa:bb:cc:dd:ee:99/position", "application/json", `{"x":1,"y":2}`, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body)
	}
	stored := deviceFromStore(t, path, "mac:aa:bb:cc:dd:ee:01")
	if stored.Pinned {
		t.Fatal("store mutated after an unknown device")
	}
	if published := deviceFromApp(app, "mac:aa:bb:cc:dd:ee:01"); published.Pinned {
		t.Fatal("published snapshot mutated after an unknown device")
	}
}

func TestDevicePositionRejectsInvalidCoordinates(t *testing.T) {
	const id = "mac:aa:bb:cc:dd:ee:01"
	cases := []struct {
		name        string
		contentType string
		body        string
	}{
		{"nan form", "application/x-www-form-urlencoded", "x=NaN&y=0"},
		{"positive infinity form", "application/x-www-form-urlencoded", "x=Inf&y=0"},
		{"negative infinity form", "application/x-www-form-urlencoded", "y=-Inf&x=0"},
		{"out of bounds x", "application/json", `{"x":1000001,"y":0}`},
		{"out of bounds negative y", "application/json", `{"x":0,"y":-1000001}`},
		{"missing y", "application/json", `{"x":1}`},
		{"missing x", "application/json", `{"y":1}`},
		{"string coordinate", "application/json", `{"x":"1","y":2}`},
		{"trailing json", "application/json", `{"x":1,"y":2}{"x":3,"y":4}`},
		{"overflowing number", "application/json", `{"x":1e999,"y":0}`},
		{"malformed json", "application/json", `{`},
		{"empty form", "application/x-www-form-urlencoded", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, app, path := newLayoutTestHandler(t, layoutDevice("192.0.2.20", "aa:bb:cc:dd:ee:01"))
			rec := doPositionRequest(handler, "/api/devices/"+id+"/position", tc.contentType, tc.body, nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body)
			}
			stored := deviceFromStore(t, path, id)
			if stored.Pinned || stored.X != 0 || stored.Y != 0 {
				t.Fatalf("store mutated on invalid input: (%v, %v) pinned=%v", stored.X, stored.Y, stored.Pinned)
			}
			published := deviceFromApp(app, id)
			if published.Pinned || published.X != 0 || published.Y != 0 {
				t.Fatalf("snapshot mutated on invalid input: (%v, %v) pinned=%v", published.X, published.Y, published.Pinned)
			}
		})
	}
}

func TestLayoutEndpointsRequireSameOrigin(t *testing.T) {
	const id = "mac:aa:bb:cc:dd:ee:01"
	cases := []struct {
		name    string
		headers map[string]string
	}{
		{"cross-origin", map[string]string{"Origin": "https://evil.example"}},
		{"cross-site fetch", map[string]string{"Sec-Fetch-Site": "cross-site"}},
		{"origin host mismatch", map[string]string{"Origin": "http://127.0.0.1:9999"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, app, path := newLayoutTestHandler(t, layoutDevice("192.0.2.20", "aa:bb:cc:dd:ee:01"))
			rec := doPositionRequest(handler, "/api/devices/"+id+"/position", "application/json", `{"x":1,"y":2}`, tc.headers)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("position status = %d, want %d: %s", rec.Code, http.StatusForbidden, rec.Body)
			}
			rec = doPositionRequest(handler, "/api/layout/reset", "", "", tc.headers)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("reset status = %d, want %d: %s", rec.Code, http.StatusForbidden, rec.Body)
			}
			stored := deviceFromStore(t, path, id)
			if stored.Pinned {
				t.Fatal("store mutated by a cross-origin request")
			}
			if published := deviceFromApp(app, id); published.Pinned {
				t.Fatal("snapshot mutated by a cross-origin request")
			}
		})
	}
}

func TestResetLayoutClearsEveryPin(t *testing.T) {
	devices := []model.Device{
		layoutDevice("192.0.2.20", "aa:bb:cc:dd:ee:01"),
		layoutDevice("192.0.2.21", "aa:bb:cc:dd:ee:02"),
	}
	handler, app, path := newLayoutTestHandler(t, devices...)
	for i, d := range devices {
		id := "mac:" + d.MAC
		rec := doPositionRequest(handler, "/api/devices/"+id+"/position", "application/json",
			`{"x":`+strconv.Itoa(i+1)+`,"y":`+strconv.Itoa(i+2)+`}`, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("pin %s status = %d, want %d: %s", id, rec.Code, http.StatusOK, rec.Body)
		}
	}

	rec := doPositionRequest(handler, "/api/layout/reset", "", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body)
	}
	for _, d := range devices {
		id := "mac:" + d.MAC
		stored := deviceFromStore(t, path, id)
		if stored.Pinned || stored.X != 0 || stored.Y != 0 {
			t.Fatalf("%s still has position (%v, %v) pinned=%v", id, stored.X, stored.Y, stored.Pinned)
		}
		published := deviceFromApp(app, id)
		if published.Pinned || published.X != 0 || published.Y != 0 {
			t.Fatalf("%s still has published position (%v, %v) pinned=%v", id, published.X, published.Y, published.Pinned)
		}
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
