package discovery

import (
	"net/netip"
	"strconv"
	"strings"
)

// parseDarwinARP reads the output of `arp -an`:
//
//	? (192.168.1.1) at 0:11:22:33:44:55 on en0 ifscope [ethernet]
//	? (192.168.1.20) at (incomplete) on en0 ifscope [ethernet]
func parseDarwinARP(data []byte) map[netip.Addr]string {
	out := make(map[netip.Addr]string)
	for _, line := range strings.Split(string(data), "\n") {
		open := strings.Index(line, "(")
		closeParen := strings.Index(line, ")")
		if open < 0 || closeParen <= open {
			continue
		}
		addr, err := netip.ParseAddr(strings.TrimSpace(line[open+1 : closeParen]))
		if err != nil || !addr.Is4() {
			continue
		}
		at := strings.Index(line[closeParen:], " at ")
		if at < 0 {
			continue
		}
		rest := line[closeParen+at+len(" at "):]
		if on := strings.Index(rest, " on "); on >= 0 {
			rest = rest[:on]
		}
		mac := normalizeMAC(rest)
		if mac == "" {
			continue
		}
		out[addr] = mac
	}
	return out
}

// parseWindowsARP reads the IPv4 rows of `arp -a`:
//
//	192.168.1.1           00-11-22-33-44-55     dynamic
func parseWindowsARP(data []byte) map[netip.Addr]string {
	out := make(map[netip.Addr]string)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		addr, err := netip.ParseAddr(fields[0])
		if err != nil || !addr.Is4() {
			continue
		}
		mac := normalizeMAC(fields[1])
		if mac == "" {
			continue
		}
		out[addr] = mac
	}
	return out
}

// parseDarwinGateway reads the gateway field of `route -n get default`.
func parseDarwinGateway(data []byte) netip.Addr {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "gateway:") {
			continue
		}
		addr, err := netip.ParseAddr(strings.TrimSpace(strings.TrimPrefix(line, "gateway:")))
		if err == nil && addr.Is4() && !addr.IsUnspecified() {
			return addr
		}
	}
	return netip.Addr{}
}

// parseWindowsGateway reads the IPv4 default route from `route print -4` and
// picks the lowest metric, matching how Windows chooses between gateways.
func parseWindowsGateway(data []byte) netip.Addr {
	var (
		best       netip.Addr
		bestMetric = int(^uint(0) >> 1)
	)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "0.0.0.0" || fields[1] != "0.0.0.0" {
			continue
		}
		addr, err := netip.ParseAddr(fields[2])
		if err != nil || !addr.Is4() || addr.IsUnspecified() {
			continue
		}
		metric, err := strconv.Atoi(fields[4])
		if err != nil {
			metric = bestMetric
		}
		if !best.IsValid() || metric < bestMetric {
			best = addr
			bestMetric = metric
		}
	}
	return best
}
