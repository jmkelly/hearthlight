package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jmkelly/hearthlight/internal/api"
	"github.com/jmkelly/hearthlight/internal/demo"
	"github.com/jmkelly/hearthlight/internal/discovery"
	"github.com/jmkelly/hearthlight/internal/model"
	"github.com/jmkelly/hearthlight/internal/store"
	"github.com/jmkelly/hearthlight/internal/traffic"
	"github.com/jmkelly/hearthlight/web"
)

var version = "dev"

// trafficInterval is how often live traffic is sampled and pushed to browsers.
const trafficInterval = 2 * time.Second

func main() {
	listen := flag.String("listen", envOr("HEARTHLIGHT_LISTEN", "127.0.0.1:8090"), "HTTP listen address")
	dataPath := flag.String("data", envOr("HEARTHLIGHT_DATA", "hearthlight-state.json"), "state file path (empty for in-memory)")
	interval := flag.Duration("interval", 60*time.Second, "scan interval")
	cidrFlag := flag.String("cidr", "", "comma-separated IPv4 CIDRs to scan (default: local LAN interfaces)")
	portsFlag := flag.String("ports", "", "extra TCP ports, e.g. 8000,9000-9010")
	timeout := flag.Duration("timeout", 350*time.Millisecond, "per-connection TCP timeout")
	concurrency := flag.Int("concurrency", 16, "parallel host probes")
	demoMode := flag.Bool("demo", false, "serve a simulated network without scanning")
	tailnetEnabled := flag.Bool("tailscale", true, "discover Tailscale peers via `tailscale status --json`")
	trafficEnabled := flag.Bool("traffic", true, "sample live traffic between this host and mapped devices")
	token := flag.String("token", envOr("HEARTHLIGHT_TOKEN", ""), "shared token required for the API (empty disables)")
	hostsFlag := flag.String("hosts", envOr("HEARTHLIGHT_HOSTS", ""), "extra hostnames allowed in the Host header, comma-separated")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("hearthlight %s\n", version)
		return
	}
	if *interval < 5*time.Second {
		fatal("interval must be at least 5s")
	}
	if *timeout < 50*time.Millisecond || *timeout > 5*time.Second {
		fatal("timeout must be between 50ms and 5s")
	}
	if *concurrency < 1 || *concurrency > 128 {
		fatal("concurrency must be between 1 and 128")
	}
	extraPorts, err := discovery.ParsePortList(*portsFlag)
	if err != nil {
		fatal("invalid --ports: %v", err)
	}

	allowedHosts := splitCSV(*hostsFlag)
	if listenHost, _, err := net.SplitHostPort(*listen); err == nil && listenHost != "" {
		allowedHosts = append(allowedHosts, listenHost)
	}
	tokenValue := strings.TrimSpace(*token)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if tokenValue != "" {
		logger.Info("API token required; open the URL with ?token=<token> once to sign in")
	} else if !listenIsLoopback(*listen) {
		logger.Warn("listening beyond loopback without an API token; anyone who can reach the port can read the network inventory")
	}
	scanRequests := make(chan struct{}, 1)
	var st *store.Store
	var layout api.LayoutStore
	var demoLayout *demo.Layout
	if *demoMode {
		demoLayout = demo.NewLayout()
		layout = demoLayout
	} else {
		st = loadState(*dataPath, logger)
		layout = st
	}
	app := api.New(scanRequests, api.Config{
		Logger:       logger,
		Token:        tokenValue,
		AllowedHosts: allowedHosts,
		Layout:       layout,
	})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *demoMode {
		logger.Info("starting in demo mode", "version", version)
		go runDemoLoop(ctx, app, scanRequests, demoLayout, logger)
		if *trafficEnabled {
			go runEvery(ctx, trafficInterval, func(now time.Time) {
				app.SetTraffic(demo.Traffic(now, trafficInterval))
			})
		}
	} else {
		app.SetSnapshot(st.Snapshot(store.SnapshotOptions{Now: time.Now(), Interval: *interval}))

		scanner, err := discovery.New(discovery.Config{
			CIDRs:       splitCSV(*cidrFlag),
			ExtraPorts:  extraPorts,
			Timeout:     *timeout,
			Concurrency: *concurrency,
			Logger:      logger,
		})
		if err != nil {
			fatal("configure scanner: %v", err)
		}
		logger.Info("starting hearthlight", "version", version, "interval", *interval, "tailscale", *tailnetEnabled)
		go runScannerLoop(ctx, app, st, scanner, *interval, *tailnetEnabled, scanRequests, logger)
		if *trafficEnabled {
			sampler := traffic.New(*tailnetEnabled, logger)
			go runEvery(ctx, trafficInterval, func(now time.Time) {
				snapshot := app.Snapshot()
				flows := sampler.Sample(ctx, snapshot.Devices, now, snapshot.Scanning)
				app.SetTraffic(model.Traffic{GeneratedAt: now, IntervalMS: trafficInterval.Milliseconds(), Flows: flows})
			})
		}
	}

	handler := app.Handler(web.Files)
	server := newServer(*listen, handler)
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http listener ready", "addr", *listen, "demo", *demoMode)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	failed := false
	select {
	case <-ctx.Done():
		logger.Info("shutting down")
	case err := <-serverErr:
		logger.Error("http server stopped", "error", err)
		failed = true
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	if failed {
		os.Exit(1)
	}
}

// newServer bounds request reads and idle keep-alives. WriteTimeout stays zero
// because it would cut long-lived SSE streams.
func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
}

// loadState opens the inventory, moving an unreadable file aside so the next
// scan starts clean and keeps saving to the configured path.
func loadState(path string, logger *slog.Logger) *store.Store {
	st, err := store.Load(path)
	if err == nil {
		return st
	}
	logger.Warn("state file is unreadable; moving it aside", "error", err, "path", path)
	backup := fmt.Sprintf("%s.corrupt-%d", path, time.Now().Unix())
	if renameErr := os.Rename(path, backup); renameErr == nil {
		logger.Warn("moved unreadable state aside", "path", backup)
	} else {
		logger.Error("move unreadable state aside", "error", renameErr)
	}
	st, err = store.Load(path)
	if err != nil {
		logger.Error("starting with in-memory state; later scans will not be saved", "error", err, "path", path)
		st, _ = store.Load("")
	}
	return st
}

func runScannerLoop(ctx context.Context, app *api.App, st *store.Store, scanner *discovery.Scanner, interval time.Duration, tailnetEnabled bool, trigger <-chan struct{}, logger *slog.Logger) {
	ssid := ""
	scan := func() {
		app.SetScanning(true)
		start := time.Now()
		result, err := scanner.Scan(ctx)
		if ctx.Err() != nil {
			return
		}
		elapsed := time.Since(start)

		lastError := ""
		if err != nil {
			lastError = err.Error()
			logger.Error("scan failed", "error", err, "duration", elapsed.Round(time.Millisecond))
		} else {
			ssid = result.SSID
			if applyErr := st.Apply(result.Devices, time.Now(), result.Networks); applyErr != nil {
				logger.Error("save state", "error", applyErr)
			}
			logger.Info("scan complete", "devices", len(result.Devices), "duration", elapsed.Round(time.Millisecond))
		}

		publish := func(scanning bool) {
			snapshot := st.Snapshot(store.SnapshotOptions{
				Now:          time.Now(),
				Interval:     interval,
				Scanning:     scanning,
				ScanDuration: elapsed,
				LastError:    lastError,
			})
			snapshot.SSID = ssid
			app.SetSnapshot(snapshot)
		}
		// Publish the LAN result before touching Tailscale so a wedged daemon
		// can never delay the local map update.
		publish(tailnetEnabled)

		// Tailnet discovery is independent: a missing binary, a stopped
		// tailscaled or malformed output only skips this reading and leaves the
		// previous tailnet state intact.
		if tailnetEnabled {
			tailnet, tailnetErr := scanner.ScanTailnet(ctx)
			if ctx.Err() != nil {
				return
			}
			if tailnetErr != nil {
				logger.Warn("tailnet discovery skipped", "error", tailnetErr)
			} else {
				if applyErr := st.ApplyTailnet(tailnet, time.Now()); applyErr != nil {
					logger.Error("save tailnet state", "error", applyErr)
				}
				logger.Info("tailnet scan complete", "tailnet", tailnet.Name, "peers", len(tailnet.Peers))
			}
			publish(false)
		}
	}

	scan()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scan()
		case <-trigger:
			scan()
		}
	}
}

func runDemoLoop(ctx context.Context, app *api.App, trigger <-chan struct{}, layout *demo.Layout, logger *slog.Logger) {
	publish := func() {
		snapshot := demo.Generate(time.Now())
		if layout != nil {
			snapshot = layout.Apply(snapshot)
		}
		app.SetSnapshot(snapshot)
	}
	publish()
	ticker := time.NewTicker(12 * time.Second)
	defer ticker.Stop()

	scan := func() {
		app.SetScanning(true)
		select {
		case <-ctx.Done():
			return
		case <-time.After(900 * time.Millisecond):
		}
		publish()
		logger.Info("demo scan complete")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publish()
		case <-trigger:
			scan()
		}
	}
}

// runEvery calls fn immediately and then on every tick until ctx ends. A slow
// call delays the next tick rather than overlapping it.
func runEvery(ctx context.Context, every time.Duration, fn func(now time.Time)) {
	fn(time.Now())
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			fn(now)
		}
	}
}

// listenIsLoopback reports whether the listen address only accepts local
// connections. An empty host means all interfaces.
func listenIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "hearthlight: "+format+"\n", args...)
	os.Exit(2)
}
