//go:build windows

package discovery

import (
	"fmt"
	"syscall"
	"unsafe"
)

var procGetExtendedTcpTable = syscall.NewLazyDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

const (
	afInet                   = 2
	tcpTableBasicConnections = 1
	errorInsufficientBuffer  = 122
)

// readTCPConnections calls GetExtendedTcpTable directly. It needs no admin
// rights and avoids spawning netstat every few seconds.
func readTCPConnections() ([]TCPConn, error) {
	if err := procGetExtendedTcpTable.Find(); err != nil {
		return nil, fmt.Errorf("GetExtendedTcpTable: %w", err)
	}
	size := uint32(16 * 1024)
	for attempt := 0; attempt < 4; attempt++ {
		buf := make([]byte, size)
		ret, _, _ := procGetExtendedTcpTable.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
			0,
			afInet,
			tcpTableBasicConnections,
			0,
		)
		switch ret {
		case 0:
			return parseWindowsTCPTable(buf), nil
		case errorInsufficientBuffer:
			// The table grew between calls; size now holds the required length.
			size += 4 * 1024
			continue
		default:
			return nil, fmt.Errorf("GetExtendedTcpTable: %w", syscall.Errno(ret))
		}
	}
	return nil, fmt.Errorf("GetExtendedTcpTable: table kept growing")
}
