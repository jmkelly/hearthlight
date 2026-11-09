package discovery

import (
	"encoding/binary"
	"net/netip"
	"strconv"
	"strings"
)

// TCPConn is one established IPv4 TCP connection on this host, owned by any
// process. Reading the table needs no privileges on Linux, macOS or Windows.
type TCPConn struct {
	Local  netip.AddrPort
	Remote netip.AddrPort
}

// ReadTCPConnections returns the established IPv4 TCP connections on this host.
func ReadTCPConnections() ([]TCPConn, error) {
	return readTCPConnections()
}

// parseProcNetTCP parses /proc/net/tcp or /proc/net/tcp6. IPv6 sockets are kept
// only when they carry an IPv4-mapped address.
func parseProcNetTCP(data []byte) []TCPConn {
	var out []TCPConn
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[3] != "01" { // 01 = ESTABLISHED
			continue
		}
		local, ok := parseProcAddrPort(fields[1])
		if !ok {
			continue
		}
		remote, ok := parseProcAddrPort(fields[2])
		if !ok {
			continue
		}
		out = append(out, TCPConn{Local: local, Remote: remote})
	}
	return out
}

func parseProcAddrPort(raw string) (netip.AddrPort, bool) {
	hexAddr, hexPort, ok := strings.Cut(raw, ":")
	if !ok {
		return netip.AddrPort{}, false
	}
	port, err := strconv.ParseUint(hexPort, 16, 16)
	if err != nil {
		return netip.AddrPort{}, false
	}
	var addr netip.Addr
	switch len(hexAddr) {
	case 8:
		addr, ok = parseProcHexAddr(hexAddr)
	case 32:
		// Four 32-bit words, each in host (little-endian) byte order.
		var b [16]byte
		for w := 0; w < 4; w++ {
			v, err := strconv.ParseUint(hexAddr[w*8:w*8+8], 16, 32)
			if err != nil {
				return netip.AddrPort{}, false
			}
			binary.LittleEndian.PutUint32(b[w*4:], uint32(v))
		}
		addr = netip.AddrFrom16(b).Unmap()
		ok = addr.Is4()
	default:
		ok = false
	}
	if !ok {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(addr, uint16(port)), true
}

// parseProcHexAddr decodes an IPv4 address written by the Linux kernel as a
// little-endian hex word, as in /proc/net/route and /proc/net/tcp.
func parseProcHexAddr(raw string) (netip.Addr, bool) {
	if len(raw) != 8 {
		return netip.Addr{}, false
	}
	v, err := strconv.ParseUint(raw, 16, 32)
	if err != nil {
		return netip.Addr{}, false
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	return netip.AddrFrom4(b), true
}

// parseDarwinNetstat parses `netstat -an -p tcp`. Addresses are written as
// a.b.c.d.port.
func parseDarwinNetstat(out []byte) []TCPConn {
	var conns []TCPConn
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || !strings.HasPrefix(fields[0], "tcp") || fields[5] != "ESTABLISHED" {
			continue
		}
		local, ok := parseDottedAddrPort(fields[3])
		if !ok {
			continue
		}
		remote, ok := parseDottedAddrPort(fields[4])
		if !ok {
			continue
		}
		conns = append(conns, TCPConn{Local: local, Remote: remote})
	}
	return conns
}

func parseDottedAddrPort(raw string) (netip.AddrPort, bool) {
	i := strings.LastIndexByte(raw, '.')
	if i <= 0 {
		return netip.AddrPort{}, false
	}
	addr, err := netip.ParseAddr(raw[:i])
	if err != nil || !addr.Is4() {
		return netip.AddrPort{}, false
	}
	port, err := strconv.ParseUint(raw[i+1:], 10, 16)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(addr, uint16(port)), true
}

// parseWindowsTCPTable decodes a MIB_TCPTABLE buffer filled by
// GetExtendedTcpTable with TCP_TABLE_BASIC_CONNECTIONS. Each row is five
// DWORDs: state, local address, local port, remote address, remote port.
// Addresses and ports are stored in network byte order.
func parseWindowsTCPTable(buf []byte) []TCPConn {
	const rowSize = 20
	const stateEstablished = 5
	if len(buf) < 4 {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(buf))
	var conns []TCPConn
	for i := 0; i < n; i++ {
		off := 4 + i*rowSize
		if off+rowSize > len(buf) {
			break
		}
		row := buf[off : off+rowSize]
		if binary.LittleEndian.Uint32(row[0:4]) != stateEstablished {
			continue
		}
		local := netip.AddrPortFrom(netip.AddrFrom4([4]byte(row[4:8])), binary.BigEndian.Uint16(row[8:10]))
		remote := netip.AddrPortFrom(netip.AddrFrom4([4]byte(row[12:16])), binary.BigEndian.Uint16(row[16:18]))
		conns = append(conns, TCPConn{Local: local, Remote: remote})
	}
	return conns
}
