//go:build darwin

package discovery

import (
	"fmt"
	"net/netip"
	"os/exec"
)

func readARPTable() (map[netip.Addr]string, error) {
	out, err := exec.Command(darwinTool("arp", "/usr/sbin/arp"), "-an").Output()
	if err != nil {
		return nil, fmt.Errorf("arp -an: %w", err)
	}
	return parseDarwinARP(out), nil
}

func defaultGateway() netip.Addr {
	out, err := exec.Command(darwinTool("route", "/sbin/route"), "-n", "get", "default").Output()
	if err != nil {
		return netip.Addr{}
	}
	return parseDarwinGateway(out)
}

// darwinTool prefers PATH but falls back to the system location so a binary
// launched from Finder still finds the network tools.
func darwinTool(name, fallback string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return fallback
}
