package discovery

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cgnatPrefix = netip.MustParsePrefix("100.64.0.0/10")

// localPrefixes returns the IPv4 LAN prefixes this host can scan, plus the
// host's own addresses. Large networks are capped to the local /24 so a typo
// or a corporate-sized subnet cannot turn into a runaway scan.
func localPrefixes() ([]netip.Prefix, map[netip.Addr]bool, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, fmt.Errorf("list interfaces: %w", err)
	}
	self := make(map[netip.Addr]bool)
	seen := make(map[netip.Prefix]struct{})

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, raw := range addrs {
			ipnet, ok := raw.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			addr, ok := netip.AddrFromSlice(ip4)
			if !ok {
				continue
			}
			addr = addr.Unmap()
			if !isScannable(addr) {
				continue
			}
			self[addr] = true

			ones, bits := ipnet.Mask.Size()
			if bits != 32 || ones <= 0 {
				continue
			}
			prefix := netip.PrefixFrom(addr, ones).Masked()
			if prefix.Bits() < 24 {
				prefix = netip.PrefixFrom(addr, 24).Masked()
			}
			seen[prefix] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil, self, fmt.Errorf("no scannable IPv4 LAN interface found")
	}
	prefixes := make([]netip.Prefix, 0, len(seen))
	for p := range seen {
		prefixes = append(prefixes, p)
	}
	sort.Slice(prefixes, func(i, j int) bool { return prefixes[i].String() < prefixes[j].String() })
	return prefixes, self, nil
}

func isVirtualInterface(name string) bool {
	name = strings.ToLower(name)
	if name == "lo" {
		return true
	}
	for _, prefix := range []string{
		"docker", "br-", "veth", "virbr", "podman", "cni", "flannel",
		"tailscale", "zt", "wg", "tun", "tap", "kube", "lxc", "lxd",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isScannable(addr netip.Addr) bool {
	return addr.Is4() &&
		!addr.IsLoopback() &&
		!addr.IsLinkLocalUnicast() &&
		!addr.IsMulticast() &&
		!addr.IsUnspecified() &&
		!cgnatPrefix.Contains(addr)
}

// enumerateHosts expands prefixes into individual addresses, excluding the
// network and broadcast addresses where they exist. The hard limit keeps a
// misconfigured explicit CIDR from scanning an entire /16.
func enumerateHosts(prefixes []netip.Prefix) ([]netip.Addr, error) {
	seen := make(map[netip.Addr]struct{})
	var hosts []netip.Addr
	for _, raw := range prefixes {
		prefix, err := netip.ParsePrefix(raw.String())
		if err != nil || !prefix.Addr().Is4() {
			return nil, fmt.Errorf("not an IPv4 prefix: %s", raw)
		}
		prefix = prefix.Masked()
		last := lastIPv4(prefix)
		network := prefix.Addr()
		for addr := network; prefix.Contains(addr); addr = addr.Next() {
			if !addr.IsValid() {
				break
			}
			if prefix.Bits() <= 30 && (addr == network || addr == last) {
				continue
			}
			if _, ok := seen[addr]; ok {
				continue
			}
			seen[addr] = struct{}{}
			hosts = append(hosts, addr)
			if len(hosts) > 2048 {
				return nil, fmt.Errorf("scan range includes more than 2048 hosts; use a smaller CIDR")
			}
		}
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Less(hosts[j]) })
	return hosts, nil
}

func lastIPv4(prefix netip.Prefix) netip.Addr {
	addr := prefix.Masked().Addr()
	b := addr.As4()
	if prefix.Bits() >= 32 {
		return addr
	}
	v := binary.BigEndian.Uint32(b[:])
	hostBits := uint(32 - prefix.Bits())
	v |= (uint32(1) << hostBits) - 1
	var out [4]byte
	binary.BigEndian.PutUint32(out[:], v)
	return netip.AddrFrom4(out)
}

// wakeNeighbors sends a single UDP datagram to each address. On Linux this
// forces the kernel to resolve the destination's ARP entry without requiring
// raw sockets or root privileges.
func wakeNeighbors(ctx context.Context, addrs []netip.Addr, timeout time.Duration) {
	sem := make(chan struct{}, 64)
	var wg sync.WaitGroup
	for _, addr := range addrs {
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
			conn, err := net.DialTimeout("udp4", net.JoinHostPort(addr.String(), "9"), timeout)
			if err != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(timeout))
			_, _ = conn.Write([]byte{0})
			_ = conn.Close()
		}(addr)
	}
	wg.Wait()
}

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

func normalizeMAC(raw string) string {
	mac := strings.ToLower(strings.TrimSpace(raw))
	if len(mac) != 17 || mac == "00:00:00:00:00:00" {
		return ""
	}
	for i, r := range mac {
		if i%3 == 2 {
			if r != ':' {
				return ""
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return ""
		}
	}
	return mac
}
