// Package api serves the JSON snapshot, an SSE stream, and the embedded UI.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

// App holds the latest snapshot and fans it out to connected browsers.
type App struct {
	mu           sync.RWMutex
	snapshot     model.Snapshot
	payload      []byte
	traffic      []byte
	clients      map[chan sseEvent]struct{}
	scanRequests chan<- struct{}
	logger       *slog.Logger
}

// sseEvent is one named Server-Sent Event.
type sseEvent struct {
	name string
	data []byte
}

func New(scanRequests chan<- struct{}, logger *slog.Logger) *App {
	if logger == nil {
		logger = slog.Default()
	}
	a := &App{
		clients:      make(map[chan sseEvent]struct{}),
		scanRequests: scanRequests,
		logger:       logger,
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

	return securityHeaders(logRequests(a.logger, mux))
}

func (a *App) handleState(w http.ResponseWriter, _ *http.Request) {
	a.mu.RLock()
	payload := a.payload
	a.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(payload)
}

func (a *App) handleScan(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
