package discovery

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
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
		// Linux containers and overlays.
		"docker", "br-", "veth", "virbr", "podman", "cni", "flannel",
		"kube", "lxc", "lxd",
		// VPN and tunnel interfaces, including macOS utun and Windows TAP.
		"tailscale", "zt", "wg", "tun", "tap", "utun", "ipsec", "ppp",
		"wireguard", "openvpn", "zerotier", "vethernet", "hyper-v",
		// VM and capture adapters.
		"vmware", "virtualbox", "vboxnet", "npcap", "wsl",
		// macOS wireless and bridge pseudo-interfaces.
		"bridge", "awdl", "llw", "anpi", "gif", "stf", "p2p",
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

// normalizeMAC accepts the formats used by Linux, macOS and Windows ARP
// tables. macOS omits leading zeroes from octets, and Windows uses dashes.
func normalizeMAC(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	sep := ":"
	if strings.Contains(raw, "-") {
		sep = "-"
	}
	parts := strings.Split(raw, sep)
	if len(parts) != 6 {
		return ""
	}
	out := make([]string, 6)
	for i, part := range parts {
		if len(part) == 0 || len(part) > 2 {
			return ""
		}
		if len(part) == 1 {
			part = "0" + part
		}
		if _, err := strconv.ParseUint(part, 16, 8); err != nil {
			return ""
		}
		out[i] = part
	}
	mac := strings.Join(out, ":")
	if mac == "00:00:00:00:00:00" || mac == "ff:ff:ff:ff:ff:ff" {
		return ""
	}
	first, _ := strconv.ParseUint(out[0], 16, 8)
	if first&1 == 1 {
		// IPv4 multicast and broadcast entries are not devices.
		return ""
	}
	return mac
}
