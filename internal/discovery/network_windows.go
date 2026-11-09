//go:build windows

package discovery

import (
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func readARPTable() (map[netip.Addr]string, error) {
	out, err := runWindowsTool("arp", "-a")
	if err != nil {
		return nil, err
	}
	return parseWindowsARP(out), nil
}

func defaultGateway() netip.Addr {
	out, err := runWindowsTool("route", "print", "-4")
	if err != nil {
		return netip.Addr{}
	}
	return parseWindowsGateway(out)
}

// runWindowsTool resolves system tools under System32 so a service account with
// a minimal PATH can still read the ARP and routing tables.
func runWindowsTool(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(windowsSystemTool(name), args...).Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

func windowsSystemTool(name string) string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	path := filepath.Join(root, "System32", name+".exe")
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	return name
}
