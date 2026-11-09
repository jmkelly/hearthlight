// Package api serves the JSON snapshot, an SSE stream, and the embedded UI.
package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

// tokenCookie is the session cookie set after a successful ?token= login.
const tokenCookie = "hearthlight_token"

// Config holds the HTTP boundary policy for the app.
type Config struct {
	Logger       *slog.Logger
	Token        string
	AllowedHosts []string
}

// App holds the latest snapshot and fans it out to connected browsers.
type App struct {
	mu           sync.RWMutex
	snapshot     model.Snapshot
	payload      []byte
	traffic      []byte
	clients      map[chan sseEvent]struct{}
	scanRequests chan<- struct{}
	logger       *slog.Logger
	token        string
	allowedHosts []string
}

// sseEvent is one named Server-Sent Event.
type sseEvent struct {
	name string
	data []byte
}

func New(scanRequests chan<- struct{}, cfg Config) *App {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	a := &App{
		clients:      make(map[chan sseEvent]struct{}),
		scanRequests: scanRequests,
		logger:       logger,
		token:        cfg.Token,
		allowedHosts: cfg.AllowedHosts,
	}
	initial := model.Snapshot{
		GeneratedAt: time.Now(),
		Networks:    []string{},
		Devices:     []model.Device{},
	}
	a.setSnapshot(initial)
	return a
}

// SetSnapshot publishes a completed scan and wakes every SSE client.
func (a *App) SetSnapshot(snapshot model.Snapshot) {
	if snapshot.GeneratedAt.IsZero() {
		snapshot.GeneratedAt = time.Now()
	}
	a.setSnapshot(snapshot)
}

func (a *App) setSnapshot(snapshot model.Snapshot) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		a.logger.Error("encode snapshot", "error", err)
		return
	}
	a.mu.Lock()
	a.snapshot = snapshot
	a.payload = payload
	a.broadcastLocked(sseEvent{name: "state", data: payload})
	a.mu.Unlock()
}

// broadcastLocked queues an event for every client. A client too slow to keep
// up misses the event; the next state or traffic event supersedes it.
func (a *App) broadcastLocked(event sseEvent) {
	for client := range a.clients {
		select {
		case client <- event:
		default:
		}
	}
}

// Snapshot returns the latest published snapshot.
func (a *App) Snapshot() model.Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.snapshot
}

// SetTraffic publishes a live traffic sample to every SSE client. Traffic is
// sent as its own small event so the full snapshot is not resent every tick.
func (a *App) SetTraffic(traffic model.Traffic) {
	if traffic.Flows == nil {
		traffic.Flows = []model.Flow{}
	}
	payload, err := json.Marshal(traffic)
	if err != nil {
		a.logger.Error("encode traffic", "error", err)
		return
	}
	a.mu.Lock()
	a.traffic = payload
	a.broadcastLocked(sseEvent{name: "traffic", data: payload})
	a.mu.Unlock()
}

// SetScanning flips the live indicator without waiting for a scan to finish.
func (a *App) SetScanning(scanning bool) {
	a.mu.Lock()
	if a.snapshot.Scanning == scanning {
		a.mu.Unlock()
		return
	}
	snapshot := a.snapshot
	snapshot.Scanning = scanning
	snapshot.GeneratedAt = time.Now()
	payload, err := json.Marshal(snapshot)
	if err != nil {
		a.mu.Unlock()
		a.logger.Error("encode snapshot", "error", err)
		return
	}
	a.snapshot = snapshot
	a.payload = payload
	a.broadcastLocked(sseEvent{name: "state", data: payload})
	a.mu.Unlock()
}

func (a *App) Handler(static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("GET /api/state", a.handleState)
	mux.HandleFunc("GET /api/events", a.handleEvents)
	mux.HandleFunc("POST /api/scan", a.handleScan)

	fileServer := http.FileServer(http.FS(static))
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	}))

	return logRequests(a.logger, securityHeaders(a.requireAllowedHost(a.requireToken(mux))))
}

// requireAllowedHost rejects a Host header that is not loopback, an IP literal
// or an explicitly configured hostname. This closes the DNS-rebinding path
// that lets another origin read the API response.
func (a *App) requireAllowedHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostAllowed(r.Host, a.allowedHosts) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireToken gates every route except /healthz behind the optional shared
// token. A valid ?token= on a page request sets a session cookie so the UI and
// its EventSource stream authenticate without putting the token in each URL.
func (a *App) requireToken(next http.Handler) http.Handler {
	if a.token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		if !tokenEqual(requestToken(r), a.token) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"error":"unauthorized"}`)
				return
			}
			http.Error(w, "unauthorized: append ?token=<token> to the URL to sign in", http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("token") != "" && !strings.HasPrefix(r.URL.Path, "/api/") {
			http.SetCookie(w, &http.Cookie{
				Name:     tokenCookie,
				Value:    a.token,
				Path:     "/",
				HttpOnly: true,
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteStrictMode,
			})
			u := *r.URL
			query := u.Query()
			query.Del("token")
			u.RawQuery = query.Encode()
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestToken(r *http.Request) string {
	if scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " "); ok && strings.EqualFold(scheme, "Bearer") {
		return strings.TrimSpace(token)
	}
	if token := r.Header.Get("X-Hearthlight-Token"); token != "" {
		return token
	}
	if cookie, err := r.Cookie(tokenCookie); err == nil {
		return cookie.Value
	}
	return r.URL.Query().Get("token")
}

// tokenEqual compares hashes so the comparison time does not depend on the
// token length or the position of the first wrong byte.
func tokenEqual(got, want string) bool {
	gotHash := sha256.Sum256([]byte(got))
	wantHash := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(gotHash[:], wantHash[:]) == 1
}

func hostAllowed(raw string, extra []string) bool {
	host, ok := normalizeHost(raw)
	if !ok {
		return false
	}
	if host == "localhost" {
		return true
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	for _, allowed := range extra {
		if normalized, ok := normalizeHost(allowed); ok && normalized == host {
			return true
		}
	}
	return false
}

func normalizeHost(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	host := raw
	if h, _, err := net.SplitHostPort(raw); err == nil {
		host = h
	} else if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		host = raw[1 : len(raw)-1]
	} else if strings.Count(raw, ":") > 1 {
		if _, err := netip.ParseAddr(raw); err != nil {
			return "", false
		}
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" {
		return "", false
	}
	return host, true
}

// sameOrigin rejects browser requests whose Origin or Sec-Fetch-Site says they
// came from another site. Non-browser clients send neither header.
func sameOrigin(r *http.Request) bool {
	switch site := r.Header.Get("Sec-Fetch-Site"); site {
	case "", "same-origin", "none":
	default:
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func (a *App) handleState(w http.ResponseWriter, _ *http.Request) {
	a.mu.RLock()
	payload := a.payload
	a.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(payload)
}

func (a *App) handleScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !sameOrigin(r) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"status":"forbidden"}`)
		return
	}
	if a.scanRequests == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"status":"unavailable"}`)
		return
	}
	a.mu.RLock()
	scanning := a.snapshot.Scanning
	a.mu.RUnlock()
	if scanning {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"status":"scanning"}`)
		return
	}
	select {
	case a.scanRequests <- struct{}{}:
	default:
	}
	w.WriteHeader(http.StatusAccepted)
	_, _ = io.WriteString(w, `{"status":"queued"}`)
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	client := make(chan sseEvent, 8)
	a.mu.Lock()
	a.clients[client] = struct{}{}
	initial := a.payload
	initialTraffic := a.traffic
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.clients, client)
		a.mu.Unlock()
	}()

	if _, err := fmt.Fprintf(w, "event: state\ndata: %s\n\n", initial); err != nil {
		return
	}
	if initialTraffic != nil {
		if _, err := fmt.Fprintf(w, "event: traffic\ndata: %s\n\n", initialTraffic); err != nil {
			return
		}
	}
	flusher.Flush()

	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-client:
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.name, event.data); err != nil {
				return
			}
			flusher.Flush()
		case <-keepalive.C:
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		if r.URL.Path == "/api/events" {
			return
		}
		logger.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start).Round(time.Millisecond),
		)
	})
}
