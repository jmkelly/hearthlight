package discovery

import (
	"encoding/binary"
	"testing"
)

func TestParseProcHexAddr(t *testing.T) {
	addr, ok := parseProcHexAddr("010200C0")
	if !ok || addr.String() != "192.0.2.1" {
		t.Fatalf("got %s ok=%v, want 192.0.2.1", addr, ok)
	}
	if _, ok := parseProcHexAddr("nothex"); ok {
		t.Fatal("expected invalid hex address to be rejected")
	}
}

func TestParseProcNetTCP(t *testing.T) {
	tcp := []byte(`  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0277 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 1 1 0000000000000000 100 0 0 10 0
   1: 3A0200C0:CC38 0A0200C0:0016 01 00000000:00000000 02:000A7B2C 00000000  1000        0 2 1 0000000000000000 20 4 30 10 -1
   2: 3A0200C0:CC39 0A0200C0:01BB 06 00000000:00000000 00:00000000 00000000  1000        0 0 3 0000000000000000
`)
	conns := parseProcNetTCP(tcp)
	if len(conns) != 1 {
		t.Fatalf("got %d conns, want 1 established: %+v", len(conns), conns)
	}
	if got := conns[0].Local.String(); got != "192.0.2.58:52280" {
		t.Errorf("local = %s", got)
	}
	if got := conns[0].Remote.String(); got != "192.0.2.10:22" {
		t.Errorf("remote = %s", got)
	}

	tcp6 := []byte(`  sl  local_address                         remote_address                        st
   0: 0000000000000000FFFF00003A0200C0:1F9A 0000000000000000FFFF00004D0200C0:D431 01 00000000:00000000
   1: 00000000000000000000000001000000:1F9A 00000000000000000000000001000000:D432 01 00000000:00000000
`)
	conns = parseProcNetTCP(tcp6)
	if len(conns) != 1 {
		t.Fatalf("got %d v4-mapped conns, want 1: %+v", len(conns), conns)
	}
	if got := conns[0].Remote.String(); got != "192.0.2.77:54321" {
		t.Errorf("mapped remote = %s", got)
	}
}

func TestParseDarwinNetstat(t *testing.T) {
	out := []byte(`Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  192.0.2.58.52344       192.0.2.10.22          ESTABLISHED
tcp4       0      0  192.0.2.58.52345       192.0.2.10.443         TIME_WAIT
tcp46      0      0  *.8090                 *.*                    LISTEN
tcp6       0      0  fe80::1%lo0.1024       fe80::1%lo0.49152      ESTABLISHED
`)
	conns := parseDarwinNetstat(out)
	if len(conns) != 1 {
		t.Fatalf("got %d conns, want 1: %+v", len(conns), conns)
	}
	if conns[0].Local.String() != "192.0.2.58:52344" || conns[0].Remote.String() != "192.0.2.10:22" {
		t.Fatalf("got %+v", conns[0])
	}
}

func TestParseWindowsTCPTable(t *testing.T) {
	row := func(state uint32, local [4]byte, lport uint16, remote [4]byte, rport uint16) []byte {
		b := make([]byte, 20)
		binary.LittleEndian.PutUint32(b[0:], state)
		copy(b[4:8], local[:])
		binary.BigEndian.PutUint16(b[8:], lport)
		copy(b[12:16], remote[:])
		binary.BigEndian.PutUint16(b[16:], rport)
		return b
	}
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 2)
	buf = append(buf, row(5, [4]byte{192, 0, 2, 58}, 50000, [4]byte{192, 0, 2, 20}, 445)...)
	buf = append(buf, row(11, [4]byte{192, 0, 2, 58}, 50001, [4]byte{192, 0, 2, 20}, 80)...)
	buf = append(buf, make([]byte, 64)...) // slack after the table is ignored

	conns := parseWindowsTCPTable(buf)
	if len(conns) != 1 {
		t.Fatalf("got %d conns, want 1: %+v", len(conns), conns)
	}
	if conns[0].Local.String() != "192.0.2.58:50000" || conns[0].Remote.String() != "192.0.2.20:445" {
		t.Fatalf("got %+v", conns[0])
	}
}

func TestParseTailnetCounters(t *testing.T) {
	data := []byte(`{
  "BackendState": "Running",
  "Self": {"ID": "nSelf", "RxBytes": 999, "TxBytes": 999},
  "Peer": {
    "nodekey:aaa": {"ID": "nPeer1", "RxBytes": 16376, "TxBytes": 20144},
    "nodekey:bbb": {"PublicKey": "nodekey:bbb", "RxBytes": 0, "TxBytes": 0}
  }
}`)
	counters, err := parseTailnetCounters(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(counters) != 2 {
		t.Fatalf("got %d counters, want 2 (self excluded): %+v", len(counters), counters)
	}
	if c := counters["nPeer1"]; c.RxBytes != 16376 || c.TxBytes != 20144 {
		t.Errorf("nPeer1 = %+v", c)
	}
	if _, ok := counters["nodekey:bbb"]; !ok {
		t.Error("peer without ID should fall back to its public key")
	}
}
