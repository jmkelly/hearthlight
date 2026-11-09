//go:build linux

package discovery

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

// readARPTable parses /proc/net/arp. Only entries with a real hardware address
// are returned.
func readARPTable() (map[netip.Addr]string, error) {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return nil, fmt.Errorf("read ARP table: %w", err)
	}
	out := make(map[netip.Addr]string)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		addr, err := netip.ParseAddr(fields[0])
		if err != nil || !addr.Is4() {
			continue
		}
		mac := normalizeMAC(fields[3])
		if mac == "" {
			continue
		}
		out[addr] = mac
	}
	return out, nil
}

func defaultGateway() netip.Addr {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return netip.Addr{}
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "00000000" {
			continue
		}
		if addr, ok := parseProcHexAddr(fields[2]); ok {
			return addr
		}
	}
	return netip.Addr{}
}

func parseProcHexAddr(raw string) (netip.Addr, bool) {
	if len(raw) != 8 {
		return netip.Addr{}, false
	}
	v, err := strconv.ParseUint(raw, 16, 32)
	if err != nil {
		return netip.Addr{}, false
	}
	b := [4]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
	return netip.AddrFrom4(b), true
}
