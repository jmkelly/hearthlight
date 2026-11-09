//go:build linux

package discovery

import (
	"fmt"
	"os"
)

func readTCPConnections() ([]TCPConn, error) {
	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return nil, fmt.Errorf("read TCP table: %w", err)
	}
	conns := parseProcNetTCP(data)
	// Dual-stack sockets list IPv4 peers as IPv4-mapped IPv6 addresses.
	if data6, err := os.ReadFile("/proc/net/tcp6"); err == nil {
		conns = append(conns, parseProcNetTCP(data6)...)
	}
	return conns, nil
}
