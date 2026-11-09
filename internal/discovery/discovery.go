package discovery

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

// Config is the discovery boundary. Values are validated here rather than in
// the scanner's inner loop.
type Config struct {
	CIDRs       []string
	ExtraPorts  []int
	Timeout     time.Duration
	Concurrency int
	Logger      *slog.Logger
}

type Scanner struct {
	prefixes    []netip.Prefix
	explicit    bool
	ports       []int
	timeout     time.Duration
	concurrency int
	logger      *slog.Logger
	http        *http.Client
}

// Result is one completed observation of the network.
type Result struct {
	Devices  []model.Device
	Networks []string
	// SSID is the current Wi-Fi network name. It is live host state, not part
	// of the inventory, so callers must not persist it.
	SSID string
}

func New(cfg Config) (*Scanner, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 350 * time.Millisecond
	}
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 16
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	s := &Scanner{
		ports:       mergePorts(cfg.ExtraPorts),
		timeout:     timeout,
		concurrency: concurrency,
		logger:      logger,
	}
	for _, raw := range cfg.CIDRs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", raw, err)
		}
		if !prefix.Addr().Is4() {
			return nil, fmt.Errorf("only IPv4 CIDRs are supported: %q", raw)
		}
		s.prefixes = append(s.prefixes, prefix.Masked())
		s.explicit = true
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 1200 * time.Millisecond}).DialContext,
		// The certificate is used only as a label for a homelab service. The
		// connection never carries user data and is not used for trust.
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
		MaxIdleConns:      10,
	}
	s.http = &http.Client{
		Transport: transport,
		Timeout:   2 * time.Second,
		// A redirect can point at any host, so following one would let the
		// scanned device steer the probe into unrelated services.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return s, nil
}

// Scan performs one full observation. ARP (populated by a UDP wake) finds the
// hosts, TCP probes identify the services, and HTTP/SSH/TLS fingerprints give
// bubbles their human names.
func (s *Scanner) Scan(ctx context.Context) (Result, error) {
	prefixes := append([]netip.Prefix(nil), s.prefixes...)
	self := make(map[netip.Addr]bool)
	var err error
	if s.explicit {
		_, self, _ = localPrefixes()
	} else {
		prefixes, self, err = localPrefixes()
		if err != nil {
			return Result{}, err
		}
	}

	hosts, err := enumerateHosts(prefixes)
	if err != nil {
		return Result{}, err
	}
	gateway := defaultGateway()
	for addr := range self {
		hosts = append(hosts, addr)
	}
	if gateway.IsValid() {
		hosts = append(hosts, gateway)
	}
	hosts = uniqueAddrs(hosts)

	networks := prefixStrings(prefixes)
	ssid := readWirelessSSID(ctx)
	s.logger.Info("scan starting", "networks", networks, "hosts", len(hosts))
	wakeNeighbors(ctx, hosts, s.timeout)

	// A TCP RST on a closed port is the most reliable unprivileged liveness
	// signal on a LAN. ARP alone misses sleeping Wi-Fi devices and is very
	// sensitive to cache timing, so use it only for MAC addresses and as an
	// additional candidate source.
	alive := s.sweepAlive(ctx, hosts)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	s.logger.Info("liveness sweep finished", "alive", len(alive))

	arp, arpErr := readARPTable()
	macs := make(map[netip.Addr]string)
	if arpErr != nil {
		s.logger.Warn("ARP table unavailable; falling back to TCP service probes", "error", arpErr)
	} else {
		hostSet := make(map[netip.Addr]struct{}, len(hosts))
		for _, host := range hosts {
			hostSet[host] = struct{}{}
		}
		for addr, mac := range arp {
			if _, ok := hostSet[addr]; ok {
				macs[addr] = mac
			}
		}
	}
	s.logger.Info("ARP table read", "entries", len(macs))

	candidates := make([]netip.Addr, 0, len(macs)+len(alive)+len(self)+1)
	for addr := range macs {
		candidates = append(candidates, addr)
	}
	for addr := range alive {
		candidates = append(candidates, addr)
	}
	for addr := range self {
		candidates = append(candidates, addr)
	}
	if gateway.IsValid() {
		candidates = append(candidates, gateway)
	}
	candidates = uniqueAddrs(candidates)
	if arpErr != nil {
		candidates = hosts
	}
	if len(candidates) == 0 {
		return Result{Networks: networks, SSID: ssid}, nil
	}

	devices := s.probeAll(ctx, candidates, macs, self, gateway, alive)
	s.logger.Info("network observation finished", "devices", len(devices))
	return Result{Devices: devices, Networks: networks, SSID: ssid}, nil
}

func (s *Scanner) sweepAlive(ctx context.Context, hosts []netip.Addr) map[netip.Addr]bool {
	concurrency := s.concurrency * 4
	if concurrency < 64 {
		concurrency = 64
	}
	sem := make(chan struct{}, concurrency)
	dialer := &net.Dialer{Timeout: s.timeout}
	var wg sync.WaitGroup
	var mu sync.Mutex
	alive := make(map[netip.Addr]bool)

	for _, host := range hosts {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(addr netip.Addr) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			conn, err := dialer.DialContext(ctx, "tcp4", net.JoinHostPort(addr.String(), "33434"))
			if err == nil {
				_ = conn.Close()
				mu.Lock()
				alive[addr] = true
				mu.Unlock()
				return
			}
			if connectionRefused(err) {
				mu.Lock()
				alive[addr] = true
				mu.Unlock()
			}
		}(host)
	}
	wg.Wait()
	return alive
}

func (s *Scanner) probeAll(ctx context.Context, hosts []netip.Addr, macs map[netip.Addr]string, self map[netip.Addr]bool, gateway netip.Addr, alive map[netip.Addr]bool) []model.Device {
	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	devices := make([]model.Device, 0, len(hosts))

	for _, host := range hosts {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(addr netip.Addr) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			device, ok := s.probeHost(ctx, addr, macs[addr], self[addr], addr == gateway, alive[addr])
			if !ok {
				return
			}
			mu.Lock()
			devices = append(devices, device)
			mu.Unlock()
		}(host)
	}
	wg.Wait()

	sort.Slice(devices, func(i, j int) bool {
		a, errA := netip.ParseAddr(devices[i].IP)
		b, errB := netip.ParseAddr(devices[j].IP)
		if errA == nil && errB == nil {
			return a.Less(b)
		}
		return devices[i].IP < devices[j].IP
	})
	return devices
}

func (s *Scanner) probeHost(ctx context.Context, addr netip.Addr, mac string, isSelf, isGateway, knownAlive bool) (model.Device, bool) {
	ip := addr.String()
	openPorts := s.scanPorts(ctx, ip)
	if len(openPorts) == 0 && !isSelf && !isGateway && !knownAlive && !s.tcpAlive(ctx, ip) {
		return model.Device{}, false
	}

	now := time.Now()
	hostname := ""
	if isSelf {
		hostname, _ = os.Hostname()
	}
	if hostname == "" {
		hostname = s.reverseLookup(ctx, ip)
	}

	services := s.identifyServices(ctx, ip, openPorts, now)
	for i := range services {
		services[i].Local = true
	}

	vendor := vendorForMAC(mac)
	return model.Device{
		ID:        deviceID(ip, mac),
		IP:        ip,
		MAC:       mac,
		Hostname:  hostname,
		Vendor:    vendor,
		Kind:      classifyDevice(isGateway, hostname, vendor, services),
		Gateway:   isGateway,
		Self:      isSelf,
		FirstSeen: now,
		LastSeen:  now,
		Local:     true,
		Online:    true,
		Services:  services,
	}, true
}

// identifyServices names and fingerprints a set of open ports. It is shared by
// LAN and tailnet probing so both paths produce the same labels.
func (s *Scanner) identifyServices(ctx context.Context, ip string, openPorts []int, now time.Time) []model.Service {
	services := make([]model.Service, 0, len(openPorts))
	for _, port := range openPorts {
		name, category := describePort(port)
		product, source := s.identify(ctx, ip, port)
		services = append(services, model.Service{
			Port:      port,
			Protocol:  "tcp",
			Name:      name,
			Product:   product,
			Category:  category,
			Source:    source,
			FirstSeen: now,
			LastSeen:  now,
			Online:    true,
		})
	}
	return services
}

func (s *Scanner) scanPorts(ctx context.Context, ip string) []int {
	sem := make(chan struct{}, 6)
	dialer := &net.Dialer{Timeout: s.timeout}
	var wg sync.WaitGroup
	var mu sync.Mutex
	open := make([]int, 0, 8)

	for _, port := range s.ports {
		port := port
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			conn, err := dialer.DialContext(ctx, "tcp4", net.JoinHostPort(ip, strconv.Itoa(port)))
			if err != nil {
				return
			}
			_ = conn.Close()
			mu.Lock()
			open = append(open, port)
			mu.Unlock()
		}()
	}
	wg.Wait()
	sort.Ints(open)
	return open
}

// tcpAlive distinguishes "host is up but exposes no known ports" from "host is
// gone". A closed port on a live host answers with RST, which is a refusal.
func (s *Scanner) tcpAlive(ctx context.Context, ip string) bool {
	dialer := &net.Dialer{Timeout: s.timeout}
	conn, err := dialer.DialContext(ctx, "tcp4", net.JoinHostPort(ip, "33434"))
	if err == nil {
		_ = conn.Close()
		return true
	}
	return connectionRefused(err)
}

// wsaECONNREFUSED is the Windows errno for a refused TCP connect. Go's
// syscall.ECONNREFUSED is an invented value on Windows, so it never matches a
// real refusal reported by the network stack.
const wsaECONNREFUSED = syscall.Errno(10061)

func connectionRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, wsaECONNREFUSED) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "connection refused") ||
		strings.Contains(message, "actively refused")
}

func (s *Scanner) reverseLookup(ctx context.Context, ip string) string {
	lookupCtx, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(lookupCtx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	name := strings.TrimSuffix(strings.TrimSpace(names[0]), ".")
	name = strings.TrimPrefix(name, "_")
	if name == "" || name == ip || strings.HasPrefix(strings.ToLower(name), "localhost") {
		return ""
	}
	return name
}

func classifyDevice(gateway bool, hostname, vendor string, services []model.Service) string {
	if gateway {
		return "router"
	}
	host := strings.ToLower(hostname)
	vend := strings.ToLower(vendor)
	hasHost := func(parts ...string) bool {
		for _, part := range parts {
			if strings.Contains(host, part) {
				return true
			}
		}
		return false
	}
	hasVendor := func(parts ...string) bool {
		for _, part := range parts {
			if strings.Contains(vend, part) {
				return true
			}
		}
		return false
	}

	ports := make(map[int]struct{}, len(services))
	for _, svc := range services {
		ports[svc.Port] = struct{}{}
	}
	has := func(port int) bool {
		_, ok := ports[port]
		return ok
	}

	switch {
	case hasVendor("synology", "qnap") || hasHost("fileserver", "nas", "truenas", "openmediavault", "diskstation"):
		return "nas"
	case vend == "hp" || hasVendor("epson", "brother", "canon") ||
		hasHost("printer", "epson", "brother", "canon", "laserjet", "officejet", "deskjet"):
		return "printer"
	case hasHost("iphone", "ipad", "phone", "android", "pixel", "galaxy"):
		return "phone"
	case has(445) || has(2049) || has(5000) || has(5001):
		return "nas"
	case has(631):
		return "printer"
	case has(32400) || has(8096) || has(554) || has(8009):
		return "media"
	case hasHost("chromecast", "appletv", "apple-tv", "firetv", "fire-tv", "roku", "sonos", "shield", "tv"):
		return "media"
	case has(1883) || has(8123):
		return "iot"
	case hasVendor("espressif", "philips hue", "xiaomi", "tuya", "sonoff") ||
		hasHost("esp_", "esp-", "esp", "yeelink", "yeelight", "hs110", "shelly", "tasmota", "miio", "plug", "bulb", "light", "rako", "rakmodule", "echo", "alexa", "amazon-"):
		return "iot"
	case has(22) || has(8006) || has(9090) || has(9100) || has(5432) || has(6379) || has(2375) || has(9200):
		return "server"
	case has(3389) || has(5900):
		return "computer"
	case hasHost("desktop", "laptop", "ubuntu", "omarchy", "macbook", "thinkpad"):
		return "computer"
	default:
		return "device"
	}
}

func uniqueAddrs(in []netip.Addr) []netip.Addr {
	seen := make(map[netip.Addr]struct{}, len(in))
	out := make([]netip.Addr, 0, len(in))
	for _, addr := range in {
		if !addr.IsValid() {
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}

func prefixStrings(prefixes []netip.Prefix) []string {
	out := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		out = append(out, prefix.String())
	}
	sort.Strings(out)
	return out
}

func deviceID(ip, mac string) string {
	if mac != "" {
		return "mac:" + mac
	}
	return "ip:" + ip
}
