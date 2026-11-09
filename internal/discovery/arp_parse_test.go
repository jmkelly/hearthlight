package discovery

import (
	"net/netip"
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"00:11:22:33:44:55", "00:11:22:33:44:55"},
		{"00-11-22-33-44-55", "00:11:22:33:44:55"},
		{"0:1:a:2b:3c:4d", "00:01:0a:2b:3c:4d"},
		{"A4-83-E7-01-02-03", "a4:83:e7:01:02:03"},
		{"00:00:00:00:00:00", ""},
		{"ff:ff:ff:ff:ff:ff", ""},
		{"01:00:5e:00:00:16", ""},
		{"00:11:22:33:44", ""},
		{"00:11:22:33:44:555", ""},
		{"00:11:22:33:44:zz", ""},
		{"(incomplete)", ""},
	}
	for _, tt := range tests {
		if got := normalizeMAC(tt.in); got != tt.want {
			t.Errorf("normalizeMAC(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseDarwinARP(t *testing.T) {
	out := []byte(`? (192.168.1.1) at 0:11:22:33:44:55 on en0 ifscope [ethernet]
? (192.168.1.20) at a4:83:e7:1:2:3 on en0 ifscope [ethernet]
? (192.168.1.30) at (incomplete) on en0 ifscope [ethernet]
? (192.168.1.255) at ff:ff:ff:ff:ff:ff on en0 ifscope [ethernet]
? (fe80::1%en0) at 0:11:22:33:44:55 on en0 ifscope [ethernet]
`)
	got := parseDarwinARP(out)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %v", len(got), got)
	}
	if got[netip.MustParseAddr("192.168.1.1")] != "00:11:22:33:44:55" {
		t.Errorf("192.168.1.1 = %q", got[netip.MustParseAddr("192.168.1.1")])
	}
	if got[netip.MustParseAddr("192.168.1.20")] != "a4:83:e7:01:02:03" {
		t.Errorf("192.168.1.20 = %q", got[netip.MustParseAddr("192.168.1.20")])
	}
}

func TestParseWindowsARP(t *testing.T) {
	out := []byte(`
Interface: 192.168.1.100 --- 0x6
  Internet Address      Physical Address      Type
  192.168.1.1           00-11-22-33-44-55     dynamic
  192.168.1.20          a4-83-e7-01-02-03     dynamic
  192.168.1.255         ff-ff-ff-ff-ff-ff     static
  224.0.0.22            01-00-5e-00-00-16     static
`)
	got := parseWindowsARP(out)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %v", len(got), got)
	}
	if got[netip.MustParseAddr("192.168.1.1")] != "00:11:22:33:44:55" {
		t.Errorf("192.168.1.1 = %q", got[netip.MustParseAddr("192.168.1.1")])
	}
	if got[netip.MustParseAddr("192.168.1.20")] != "a4:83:e7:01:02:03" {
		t.Errorf("192.168.1.20 = %q", got[netip.MustParseAddr("192.168.1.20")])
	}
}

func TestParseDarwinGateway(t *testing.T) {
	out := []byte(`   route to: default
destination: default
       mask: default
    gateway: 192.168.1.1
  interface: en0
      flags: <UP,GATEWAY,DONE,STATIC,PRCLONING,GLOBAL>
`)
	if got := parseDarwinGateway(out); got.String() != "192.168.1.1" {
		t.Fatalf("parseDarwinGateway = %s, want 192.168.1.1", got)
	}
}

func TestParseWindowsGateway(t *testing.T) {
	out := []byte(`===========================================================================
IPv4 Route Table
===========================================================================
Active Routes:
Network Destination        Netmask          Gateway       Interface  Metric
          0.0.0.0          0.0.0.0      192.168.1.1     192.168.1.100     25
          0.0.0.0          0.0.0.0       10.8.0.1       10.8.0.2          5
        192.168.1.0    255.255.255.0         On-link      192.168.1.100    281
===========================================================================
`)
	if got := parseWindowsGateway(out); got.String() != "10.8.0.1" {
		t.Fatalf("parseWindowsGateway = %s, want lowest-metric 10.8.0.1", got)
	}
}
