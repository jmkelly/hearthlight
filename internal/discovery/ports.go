package discovery

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// defaultPorts is a compact homelab-oriented TCP port list. UDP services
// (mDNS, SSDP, WireGuard) are intentionally out of scope for this first pass.
var defaultPorts = []int{
	22, 53, 80, 139, 443, 445, 548, 554, 631, 873, 1883, 2049, 2375, 3000,
	3306, 3389, 5000, 5055, 5432, 5900, 6379, 6767, 7878, 8000, 8006, 8008,
	8009, 8080, 8081, 8090, 8096, 8123, 8384, 8443, 8686, 8883, 8888, 8989,
	9000, 9090, 9100, 9200, 9443, 9696, 32400, 62078,
}

type portInfo struct {
	name     string
	category string
}

var portCatalog = map[int]portInfo{
	22:    {"SSH", "remote"},
	53:    {"DNS", "infra"},
	80:    {"HTTP", "web"},
	139:   {"NetBIOS", "data"},
	443:   {"HTTPS", "web"},
	445:   {"SMB", "data"},
	548:   {"AFP", "data"},
	554:   {"RTSP", "media"},
	631:   {"IPP", "other"},
	873:   {"rsync", "data"},
	1883:  {"MQTT", "iot"},
	2049:  {"NFS", "data"},
	2375:  {"Docker", "infra"},
	3000:  {"HTTP", "web"},
	3306:  {"MySQL", "data"},
	3389:  {"RDP", "remote"},
	5000:  {"HTTP", "web"},
	5055:  {"Overseerr", "web"},
	5432:  {"PostgreSQL", "data"},
	5900:  {"VNC", "remote"},
	6379:  {"Redis", "data"},
	6767:  {"Bazarr", "web"},
	7878:  {"Radarr", "web"},
	8000:  {"HTTP", "web"},
	8006:  {"Proxmox", "infra"},
	8008:  {"Cast", "media"},
	8009:  {"Cast", "media"},
	8080:  {"HTTP", "web"},
	8081:  {"HTTP", "web"},
	8090:  {"Hearthlight", "web"},
	8096:  {"Jellyfin", "media"},
	8123:  {"Home Assistant", "iot"},
	8384:  {"Syncthing", "web"},
	8443:  {"HTTPS", "web"},
	8686:  {"Lidarr", "web"},
	8883:  {"MQTT/TLS", "iot"},
	8888:  {"HTTP", "web"},
	8989:  {"Sonarr", "web"},
	9000:  {"HTTP", "web"},
	9090:  {"Prometheus", "infra"},
	9100:  {"RAW", "other"},
	9200:  {"Elasticsearch", "data"},
	9443:  {"Portainer HTTPS", "infra"},
	9696:  {"Prowlarr", "web"},
	32400: {"Plex", "media"},
	62078: {"iPhone sync", "other"},
}

// ParsePortList accepts forms like "80,443,8000-8010". An empty string yields
// no extra ports.
func ParsePortList(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	seen := make(map[int]struct{})
	var ports []int
	add := func(p int) {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		ports = append(ports, p)
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty entry in port list")
		}
		if strings.Contains(part, "-") {
			loRaw, hiRaw, ok := strings.Cut(part, "-")
			if !ok || strings.Contains(hiRaw, "-") {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			lo, err := parsePort(loRaw)
			if err != nil {
				return nil, err
			}
			hi, err := parsePort(hiRaw)
			if err != nil {
				return nil, err
			}
			if lo > hi {
				return nil, fmt.Errorf("port range %q is backwards", part)
			}
			if hi-lo > 1024 {
				return nil, fmt.Errorf("port range %q is too large", part)
			}
			for p := lo; p <= hi; p++ {
				add(p)
			}
			continue
		}
		p, err := parsePort(part)
		if err != nil {
			return nil, err
		}
		add(p)
	}
	if len(ports) > 1024 {
		return nil, fmt.Errorf("more than 1024 ports requested")
	}
	sort.Ints(ports)
	return ports, nil
}

func parsePort(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	p, err := strconv.Atoi(raw)
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("invalid port %q", raw)
	}
	return p, nil
}

func mergePorts(extra []int) []int {
	set := make(map[int]struct{}, len(defaultPorts)+len(extra))
	for _, p := range defaultPorts {
		set[p] = struct{}{}
	}
	for _, p := range extra {
		if p > 0 && p <= 65535 {
			set[p] = struct{}{}
		}
	}
	ports := make([]int, 0, len(set))
	for p := range set {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports
}

func describePort(port int) (name, category string) {
	if info, ok := portCatalog[port]; ok {
		return info.name, info.category
	}
	return fmt.Sprintf("TCP %d", port), "other"
}
