package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

// tailscaleStatusTimeout bounds the `tailscale status --json` call. A wedged
// tailscaled must never stall the LAN scan.
const tailscaleStatusTimeout = 5 * time.Second

// tailscaleBinaryOverride lets tests point at a fake CLI.
var tailscaleBinaryOverride string

type tailscaleStatus struct {
	BackendState   string                    `json:"BackendState"`
	MagicDNSSuffix string                    `json:"MagicDNSSuffix"`
	CurrentTailnet *tailscaleCurrentTailnet  `json:"CurrentTailnet"`
	Self           *tailscaleNode            `json:"Self"`
	Peer           map[string]*tailscaleNode `json:"Peer"`
}

type tailscaleCurrentTailnet struct {
	Name           string `json:"Name"`
	MagicDNSSuffix string `json:"MagicDNSSuffix"`
}

type tailscaleNode struct {
	ID           string    `json:"ID"`
	PublicKey    string    `json:"PublicKey"`
	HostName     string    `json:"HostName"`
	DNSName      string    `json:"DNSName"`
	OS           string    `json:"OS"`
	TailscaleIPs []string  `json:"TailscaleIPs"`
	Online       bool      `json:"Online"`
	Tags         []string  `json:"Tags"`
	LastSeen     time.Time `json:"LastSeen"`
}

// ScanTailnet reads the local Tailscale status and probes every online peer
// over its tailnet IPv4 address. It returns an error when Tailscale is absent
// or misbehaving; callers log it and keep the previous tailnet state.
func (s *Scanner) ScanTailnet(ctx context.Context) (model.TailnetObservation, error) {
	obs, err := s.readTailnetStatus(ctx)
	if err != nil {
		return model.TailnetObservation{}, err
	}
	s.probeTailnetPeers(ctx, obs.Peers)
	return obs, nil
}

func (s *Scanner) readTailnetStatus(ctx context.Context) (model.TailnetObservation, error) {
	binary, err := tailscaleBinary()
	if err != nil {
		return model.TailnetObservation{}, err
	}
	statusCtx, cancel := context.WithTimeout(ctx, tailscaleStatusTimeout)
	defer cancel()

	cmd := exec.CommandContext(statusCtx, binary, "status", "--json")
	// Kill the CLI at the deadline and do not let a grandchild holding the
	// pipes keep Wait blocked past it.
	cmd.WaitDelay = 2 * time.Second
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		switch {
		case errors.Is(statusCtx.Err(), context.DeadlineExceeded):
			return model.TailnetObservation{}, fmt.Errorf("tailscale status timed out after %s", tailscaleStatusTimeout)
		case errors.Is(ctx.Err(), context.Canceled):
			return model.TailnetObservation{}, ctx.Err()
		default:
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return model.TailnetObservation{}, fmt.Errorf("tailscale status: %s", truncateText(msg, 240))
		}
	}
	obs, err := parseTailnetStatus(stdout.Bytes())
	if err != nil {
		return model.TailnetObservation{}, err
	}
	return obs, nil
}

// parseTailnetStatus converts `tailscale status --json` output into the wire
// observation. Peer IDs prefer the stable Tailscale node ID, then the public
// key, then the map key.
func parseTailnetStatus(data []byte) (model.TailnetObservation, error) {
	var status tailscaleStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return model.TailnetObservation{}, fmt.Errorf("decode tailscale status: %w", err)
	}
	if status.BackendState != "" && !strings.EqualFold(status.BackendState, "Running") {
		return model.TailnetObservation{}, fmt.Errorf("tailscaled backend is %q", status.BackendState)
	}

	name := ""
	if status.CurrentTailnet != nil {
		name = strings.TrimSpace(status.CurrentTailnet.Name)
	}
	suffix := strings.TrimSpace(status.MagicDNSSuffix)
	if suffix == "" && status.CurrentTailnet != nil {
		suffix = strings.TrimSpace(status.CurrentTailnet.MagicDNSSuffix)
	}
	obs := model.TailnetObservation{
		Tailnet: model.Tailnet{Name: name, MagicDNSSuffix: suffix},
	}

	seen := make(map[string]struct{})
	add := func(node *tailscaleNode, fallbackID string, self bool) {
		if node == nil {
			return
		}
		id := strings.TrimSpace(node.ID)
		if id == "" {
			id = strings.TrimSpace(node.PublicKey)
		}
		if id == "" {
			id = strings.TrimSpace(fallbackID)
		}
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		obs.Peers = append(obs.Peers, model.TailnetPeer{
			NodeID:   id,
			HostName: strings.TrimSpace(node.HostName),
			DNSName:  strings.TrimSpace(node.DNSName),
			OS:       strings.TrimSpace(node.OS),
			IP:       tailnetIPv4(node.TailscaleIPs),
			Tags:     append([]string(nil), node.Tags...),
			Online:   self || node.Online,
			Self:     self,
			LastSeen: node.LastSeen,
		})
	}
	add(status.Self, "", true)
	keys := make([]string, 0, len(status.Peer))
	for key := range status.Peer {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		add(status.Peer[key], key, false)
	}

	if len(obs.Peers) == 0 {
		return obs, fmt.Errorf("tailscale status has no self node (tailscaled not running?)")
	}
	for i := range obs.Peers {
		obs.Peers[i].Kind = tailnetKind(obs.Peers[i])
	}
	return obs, nil
}

// probeTailnetPeers probes online peers in parallel, reusing the scanner's
// port list, timeout and concurrency. A failed probe simply leaves the peer
// without services; Tailscale status still decides whether it is online.
func (s *Scanner) probeTailnetPeers(ctx context.Context, peers []model.TailnetPeer) {
	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	for i := range peers {
		peer := &peers[i]
		if !peer.Online || peer.IP == "" {
			continue
		}
		if ctx.Err() != nil {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			openPorts := s.scanPorts(ctx, peer.IP)
			now := time.Now()
			services := s.identifyServices(ctx, peer.IP, openPorts, now)
			for i := range services {
				services[i].Tailnet = true
			}
			peer.Services = services
			peer.Kind = tailnetKind(*peer)
		}()
	}
	wg.Wait()
}

// tailnetKind classifies a peer from its Tailscale OS label first, because an
// SSH port on a laptop should not turn it into a server.
func tailnetKind(peer model.TailnetPeer) string {
	switch strings.ToLower(peer.OS) {
	case "ios", "ipados", "android":
		return "phone"
	case "macos", "windows":
		return "computer"
	case "tvos", "roku", "webos":
		return "media"
	case "linux":
		kind := classifyDevice(false, peer.HostName, "", peer.Services)
		if kind == "device" || kind == "phone" {
			return "computer"
		}
		return kind
	}
	return ""
}

func tailnetIPv4(addrs []string) string {
	for _, raw := range addrs {
		addr, err := netip.ParseAddr(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		addr = addr.Unmap()
		if addr.Is4() && cgnatPrefix.Contains(addr) {
			return addr.String()
		}
	}
	return ""
}

// tailscaleBinary finds the CLI. The NixOS module puts the binary on PATH, but
// a desktop install or a systemd unit without PATH also needs a few fallbacks.
func tailscaleBinary() (string, error) {
	if tailscaleBinaryOverride != "" {
		return tailscaleBinaryOverride, nil
	}
	if override := strings.TrimSpace(os.Getenv("HEARTHLIGHT_TAILSCALE")); override != "" {
		path, err := exec.LookPath(override)
		if err != nil {
			return "", fmt.Errorf("tailscale binary %q: %w", override, err)
		}
		return path, nil
	}
	if path, err := exec.LookPath("tailscale"); err == nil {
		return path, nil
	}
	for _, path := range []string{
		"/run/current-system/sw/bin/tailscale",
		"/usr/local/bin/tailscale",
		"/usr/bin/tailscale",
		"/opt/homebrew/bin/tailscale",
		"/Applications/Tailscale.app/Contents/MacOS/Tailscale",
		`C:\Program Files\Tailscale\tailscale.exe`,
		`C:\Program Files (x86)\Tailscale\tailscale.exe`,
	} {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", errors.New("tailscale binary not found; tailnet discovery disabled")
}

func truncateText(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "…"
}
