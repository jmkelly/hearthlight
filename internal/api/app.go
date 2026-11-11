// Package api serves the JSON snapshot, an SSE stream, and the embedded UI.
package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
	"github.com/jmkelly/hearthlight/internal/store"
)

// tokenCookie is the session cookie set after a successful ?token= login.
const tokenCookie = "hearthlight_token"

// LayoutStore persists manual device positions.
type LayoutStore interface {
	SetDevicePosition(id string, x, y float64) error
	ClearDevicePositions() error
}

// Config holds the HTTP boundary policy for the app.
type Config struct {
	Logger       *slog.Logger
	Token        string
	AllowedHosts []string
	Layout       LayoutStore
	// OmarchyThemeFile is the normalized Omarchy palette file to read. An empty
	// path disables the read-only theme source.
	OmarchyThemeFile string
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
	layout       LayoutStore

	omarchyFile      string
	omarchy          []byte
	omarchySource    []byte
	omarchyAvailable bool
}

// sseEvent is one named Server-Sent Event.
type sseEvent struct {
	name string
	data []byte
}

// omarchyThemeResponse is the read-only theme source returned to browsers. The
// palette file is written by the Omarchy plugin; Hearthlight only reads it.
type omarchyThemeResponse struct {
	Available     bool              `json:"available"`
	SchemaVersion int               `json:"schemaVersion,omitempty"`
	Name          string            `json:"name,omitempty"`
	DisplayName   string            `json:"displayName,omitempty"`
	Mode          string            `json:"mode,omitempty"`
	Generation    int64             `json:"generation,omitempty"`
	ThemeID       string            `json:"themeId,omitempty"`
	Colors        map[string]string `json:"colors,omitempty"`
}

var (
	unavailableOmarchyTheme = []byte(`{"available":false}`)
	omarchyColorPattern     = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	omarchyColorNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
)

const (
	// maxSSEClients bounds concurrent event streams so one host cannot exhaust
	// the daemon's file descriptors and goroutines.
	maxSSEClients = 100
	// sseWriteTimeout bounds each stream write so a client that stops reading is
	// dropped instead of blocking the handler forever.
	sseWriteTimeout = 10 * time.Second
	// maxPosition bounds a manual map coordinate. A bad client must not store
	// values that overflow the canvas or the state file.
	maxPosition = 1e6
	// maxPositionBody bounds the request body the layout endpoints read.
	maxPositionBody = 64 << 10
	// omarchyPollInterval is how often the plugin's normalized palette file is
	// reread. Atomic renames change the file in place, so polling the content
	// is more reliable than trusting a timestamp.
	omarchyPollInterval = time.Second
	// maxOmarchyFileSize bounds the local palette file the watcher reads.
	maxOmarchyFileSize = 1 << 20
)

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
		layout:       cfg.Layout,
		omarchyFile:  cfg.OmarchyThemeFile,
		omarchy:      unavailableOmarchyTheme,
	}
	initial := model.Snapshot{
		GeneratedAt: time.Now(),
		Networks:    []string{},
		Devices:     []model.Device{},
	}
	a.setSnapshot(initial)
	if a.omarchyFile != "" {
		a.refreshOmarchy()
	}
	return a
}

// WatchOmarchy rereads the normalized palette file until ctx ends. The file is
// written by the Omarchy plugin, so a missing or malformed file only makes the
// theme source unavailable; it never stops the server.
func (a *App) WatchOmarchy(ctx context.Context) {
	if a.omarchyFile == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(omarchyPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.refreshOmarchy()
			}
		}
	}()
}

func (a *App) refreshOmarchy() {
	data, err := readOmarchyFile(a.omarchyFile)
	if err != nil {
		a.mu.Lock()
		changed := a.omarchyAvailable || a.omarchySource != nil
		if changed {
			a.omarchy = unavailableOmarchyTheme
			a.omarchySource = nil
			a.omarchyAvailable = false
			a.broadcastLocked(sseEvent{name: "omarchy", data: unavailableOmarchyTheme})
		}
		a.mu.Unlock()
		if changed {
			a.logger.Info("omarchy theme file unavailable", "path", a.omarchyFile, "error", err)
		}
		return
	}

	a.mu.RLock()
	unchanged := bytes.Equal(data, a.omarchySource)
	a.mu.RUnlock()
	if unchanged {
		return
	}

	theme, err := parseOmarchyTheme(data)
	if err != nil {
		a.mu.Lock()
		changed := a.omarchyAvailable || !bytes.Equal(a.omarchySource, data)
		a.omarchySource = data
		a.omarchy = unavailableOmarchyTheme
		a.omarchyAvailable = false
		if changed {
			a.broadcastLocked(sseEvent{name: "omarchy", data: unavailableOmarchyTheme})
		}
		a.mu.Unlock()
		a.logger.Warn("omarchy theme file is malformed", "path", a.omarchyFile, "error", err)
		return
	}

	payload, err := json.Marshal(theme)
	if err != nil {
		a.logger.Error("encode omarchy theme", "error", err)
		return
	}
	a.mu.Lock()
	a.omarchySource = data
	a.omarchy = payload
	a.omarchyAvailable = true
	a.broadcastLocked(sseEvent{name: "omarchy", data: payload})
	a.mu.Unlock()
}

func readOmarchyFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxOmarchyFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxOmarchyFileSize {
		return nil, fmt.Errorf("theme file exceeds %d bytes", maxOmarchyFileSize)
	}
	return data, nil
}

// parseOmarchyTheme accepts only the normalized palette shape. Unknown fields
// are ignored and colour values must be hex, so theme content is data and never
// code.
func parseOmarchyTheme(data []byte) (omarchyThemeResponse, error) {
	var theme omarchyThemeResponse
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&theme); err != nil {
		return omarchyThemeResponse{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return omarchyThemeResponse{}, errors.New("theme file has trailing data")
	}
	theme.Name = strings.TrimSpace(theme.Name)
	if theme.Name == "" {
		return omarchyThemeResponse{}, errors.New("theme name is missing")
	}
	colors := make(map[string]string, len(theme.Colors))
	for name, value := range theme.Colors {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if !omarchyColorNamePattern.MatchString(name) || !omarchyColorPattern.MatchString(value) {
			continue
		}
		colors[name] = value
	}
	if _, ok := colors["background"]; !ok {
		return omarchyThemeResponse{}, errors.New("background color is missing")
	}
	if _, ok := colors["foreground"]; !ok {
		return omarchyThemeResponse{}, errors.New("foreground color is missing")
	}
	theme.Colors = colors
	if theme.Mode != "dark" && theme.Mode != "light" {
		theme.Mode = ""
	}
	theme.Available = true
	return theme, nil
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
	mux.HandleFunc("GET /api/themes", a.handleThemes(static))
	mux.HandleFunc("GET /api/theme/omarchy", a.handleOmarchyTheme)
	mux.HandleFunc("GET /api/events", a.handleEvents)
	mux.HandleFunc("POST /api/scan", a.handleScan)
	mux.HandleFunc("POST /api/devices/{id}/position", a.handleDevicePosition)
	mux.HandleFunc("POST /api/layout/reset", a.handleResetLayout)

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

func (a *App) handleThemes(static fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if static == nil {
			a.logger.Error("read theme catalog", "error", "static filesystem is nil")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":"theme catalog unavailable"}`)
			return
		}
		payload, err := fs.ReadFile(static, "themes.json")
		if err != nil {
			a.logger.Error("read theme catalog", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":"theme catalog unavailable"}`)
			return
		}
		_, _ = w.Write(payload)
	}
}

// handleOmarchyTheme serves the last palette the watcher accepted. A missing or
// malformed file is reported as unavailable; the browser keeps its last theme.
func (a *App) handleOmarchyTheme(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	a.mu.RLock()
	payload := a.omarchy
	a.mu.RUnlock()
	if len(payload) == 0 {
		payload = unavailableOmarchyTheme
	}
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

func (a *App) handleDevicePosition(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if !sameOrigin(r) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"status":"forbidden"}`)
		return
	}
	if a.layout == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"status":"unavailable"}`)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPositionBody)
	x, y, ok := decodePosition(r)
	if !ok || !validPosition(x, y) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"x and y must be finite numbers within +/-1000000"}`)
		return
	}

	id := r.PathValue("id")
	if err := a.layout.SetDevicePosition(id, x, y); err != nil {
		if errors.Is(err, store.ErrUnknownDevice) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"unknown device"}`)
			return
		}
		a.logger.Error("set device position", "id", id, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":"save position"}`)
		return
	}
	a.updatePositions(func(devices []model.Device) {
		for i := range devices {
			if devices[i].ID == id {
				devices[i].X = x
				devices[i].Y = y
				devices[i].Pinned = true
			}
		}
	})
	_, _ = io.WriteString(w, `{"status":"pinned"}`)
}

func (a *App) handleResetLayout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if !sameOrigin(r) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"status":"forbidden"}`)
		return
	}
	if a.layout == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"status":"unavailable"}`)
		return
	}
	if err := a.layout.ClearDevicePositions(); err != nil {
		a.logger.Error("clear device positions", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":"reset layout"}`)
		return
	}
	a.updatePositions(func(devices []model.Device) {
		for i := range devices {
			devices[i].X = 0
			devices[i].Y = 0
			devices[i].Pinned = false
		}
	})
	_, _ = io.WriteString(w, `{"status":"reset"}`)
}

// decodePosition accepts a JSON object with numeric x and y, or a form with
// the same fields. A body without a JSON content type is probed both ways so
// simple clients are not punished for omitting the header.
func decodePosition(r *http.Request) (float64, float64, bool) {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
		return decodeJSONPosition(r.Body)
	}
	if err := r.ParseForm(); err == nil {
		rawX, rawY := r.Form.Get("x"), r.Form.Get("y")
		if rawX != "" && rawY != "" {
			x, errX := strconv.ParseFloat(rawX, 64)
			y, errY := strconv.ParseFloat(rawY, 64)
			if errX != nil || errY != nil {
				return 0, 0, false
			}
			return x, y, true
		}
	}
	return decodeJSONPosition(r.Body)
}

func decodeJSONPosition(body io.Reader) (float64, float64, bool) {
	var payload struct {
		X *float64 `json:"x"`
		Y *float64 `json:"y"`
	}
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&payload); err != nil {
		return 0, 0, false
	}
	if payload.X == nil || payload.Y == nil {
		return 0, 0, false
	}
	// Reject trailing tokens so a malformed body cannot be half-read.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return 0, 0, false
	}
	return *payload.X, *payload.Y, true
}

func validPosition(x, y float64) bool {
	return finiteInBounds(x) && finiteInBounds(y)
}

func finiteInBounds(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= maxPosition
}

// updatePositions applies a layout change to the published snapshot and wakes
// every SSE client so the map redraws without waiting for the next scan.
func (a *App) updatePositions(change func([]model.Device)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	snapshot := a.snapshot
	snapshot.Devices = append([]model.Device(nil), snapshot.Devices...)
	change(snapshot.Devices)
	payload, err := json.Marshal(snapshot)
	if err != nil {
		a.logger.Error("encode snapshot", "error", err)
		return
	}
	a.snapshot = snapshot
	a.payload = payload
	a.broadcastLocked(sseEvent{name: "state", data: payload})
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	a.mu.Lock()
	if len(a.clients) >= maxSSEClients {
		a.mu.Unlock()
		a.logger.Warn("sse client limit reached", "limit", maxSSEClients, "remote", r.RemoteAddr)
		http.Error(w, "too many event stream clients", http.StatusServiceUnavailable)
		return
	}
	client := make(chan sseEvent, 8)
	a.clients[client] = struct{}{}
	initial := a.payload
	initialTraffic := a.traffic
	var initialOmarchy []byte
	if a.omarchyAvailable {
		initialOmarchy = a.omarchy
	}
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.clients, client)
		a.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Every write gets its own deadline so a client that stops draining its
	// socket fails the write instead of holding the handler forever.
	controller := http.NewResponseController(w)
	write := func(format string, args ...any) error {
		if err := controller.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil {
			a.logger.Debug("sse write deadline unsupported", "error", err)
		}
		if _, err := fmt.Fprintf(w, format, args...); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := write("event: state\ndata: %s\n\n", initial); err != nil {
		return
	}
	if initialTraffic != nil {
		if err := write("event: traffic\ndata: %s\n\n", initialTraffic); err != nil {
			return
		}
	}
	if initialOmarchy != nil {
		if err := write("event: omarchy\ndata: %s\n\n", initialOmarchy); err != nil {
			return
		}
	}

	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-client:
			if err := write("event: %s\ndata: %s\n\n", event.name, event.data); err != nil {
				return
			}
		case <-keepalive.C:
			if err := write(": keepalive\n\n"); err != nil {
				return
			}
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

// Unwrap lets http.ResponseController reach the connection's write deadline
// through the logging middleware.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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
