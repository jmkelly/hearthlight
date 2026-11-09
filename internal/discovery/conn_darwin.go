//go:build darwin

package discovery

import (
	"fmt"
	"os/exec"
)

func readTCPConnections() ([]TCPConn, error) {
	out, err := exec.Command(darwinTool("netstat", "/usr/sbin/netstat"), "-an", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("netstat -an -p tcp: %w", err)
	}
	return parseDarwinNetstat(out), nil
}
