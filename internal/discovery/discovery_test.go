package discovery

import (
	"net/netip"
	"testing"

	"github.com/jmkelly/hearthlight/internal/model"
)

func TestParsePortList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int
		ok   bool
	}{
		{name: "empty", in: "", ok: true},
		{name: "single", in: "80", want: []int{80}, ok: true},
		{name: "list and range", in: "443, 8000-8002,80", want: []int{80, 443, 8000, 8001, 8002}, ok: true},
		{name: "deduplicates", in: "80,80,80", want: []int{80}, ok: true},
		{name: "zero", in: "0", ok: false},
		{name: "too high", in: "65536", ok: false},
		{name: "reversed", in: "90-80", ok: false},
		{name: "garbage", in: "http", ok: false},
		{name: "empty entry", in: "80,,443", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePortList(tt.in)
			if (err == nil) != tt.ok {
				t.Fatalf("ParsePortList(%q) error = %v, want ok=%v", tt.in, err, tt.ok)
			}
			if !tt.ok {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParsePortList(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ParsePortList(%q) = %v, want %v", tt.in, got, tt.want)
				}
			}
		})
	}
}

func TestEnumerateHosts(t *testing.T) {
	hosts, err := enumerateHosts([]netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")})
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 254 {
		t.Fatalf("got %d hosts, want 254", len(hosts))
	}
	if hosts[0].String() != "192.0.2.1" || hosts[len(hosts)-1].String() != "192.0.2.254" {
		t.Fatalf("unexpected host range %s..%s", hosts[0], hosts[len(hosts)-1])
	}

	two, err := enumerateHosts([]netip.Prefix{netip.MustParsePrefix("10.0.0.1/31")})
	if err != nil {
		t.Fatal(err)
	}
	if len(two) != 2 {
		t.Fatalf("got %d hosts for /31, want 2", len(two))
	}

	if _, err := enumerateHosts([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/16")}); err == nil {
		t.Fatal("expected /16 to be rejected")
	}
}

func TestParseProcHexAddr(t *testing.T) {
	addr, ok := parseProcHexAddr("010200C0")
	if !ok || addr.String() != "192.0.2.1" {
		t.Fatalf("got %s ok=%v, want 192.0.2.1", addr, ok)
	}
	if _, ok := parseProcHexAddr("nothex"); ok {
		t.Fatal("expected invalid hex address to be rejected")
	}
}

func TestClassifyDevice(t *testing.T) {
	services := func(ports ...int) []model.Service {
		out := make([]model.Service, 0, len(ports))
		for _, port := range ports {
			out = append(out, model.Service{Port: port})
		}
		return out
	}
	tests := []struct {
		name     string
		gateway  bool
		hostname string
		vendor   string
		ports    []int
		want     string
	}{
		{name: "gateway wins", gateway: true, ports: []int{22, 80}, want: "router"},
		{name: "printer", ports: []int{631, 9100}, want: "printer"},
		{name: "nas", ports: []int{22, 445}, want: "nas"},
		{name: "nas with print server", ports: []int{22, 139, 443, 445, 631, 2049}, want: "nas"},
		{name: "nas by hostname", hostname: "fileserver.lan", ports: []int{631}, want: "nas"},
		{name: "nas by vendor", vendor: "QNAP", ports: []int{631}, want: "nas"},
		{name: "printer by hostname", hostname: "epsonb218b6.lan", want: "printer"},
		{name: "media", ports: []int{32400}, want: "media"},
		{name: "cast device", ports: []int{8009, 8443, 9000}, want: "media"},
		{name: "phone", hostname: "iphone.lan", want: "phone"},
		{name: "android phone", hostname: "android-e496a047055dea61.lan", want: "phone"},
		{name: "smart plug", hostname: "hs110.lan", want: "iot"},
		{name: "lighting module", hostname: "rakmodule_75944d.lan", want: "iot"},
		{name: "smart bulb", hostname: "yeelink-light-color1_miio57604981.lan", want: "iot"},
		{name: "esp device", hostname: "esp_4b96b6.lan", want: "iot"},
		{name: "iot", ports: []int{8123}, want: "iot"},
		{name: "server", ports: []int{22, 5432}, want: "server"},
		{name: "computer", ports: []int{3389}, want: "computer"},
		{name: "desktop by hostname", hostname: "ubuntu.lan", ports: []int{8080}, want: "computer"},
		{name: "no services", ports: nil, want: "device"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyDevice(tt.gateway, tt.hostname, tt.vendor, services(tt.ports...)); got != tt.want {
				t.Fatalf("classifyDevice = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanTitle(t *testing.T) {
	got := cleanTitle("  <b>Grafana</b> &amp; friends\n")
	if got != "Grafana & friends" {
		t.Fatalf("cleanTitle = %q", got)
	}
	if !genericTitle("Welcome to nginx!") {
		t.Fatal("expected generic nginx title to be filtered")
	}
	if genericTitle("Grafana") {
		t.Fatal("did not expect Grafana title to be filtered")
	}
}

func TestLastIPv4(t *testing.T) {
	tests := map[string]string{
		"192.0.2.0/24":    "192.0.2.255",
		"10.0.0.0/30":     "10.0.0.3",
		"192.168.1.10/32": "192.168.1.10",
		"172.16.5.0/23":   "172.16.5.255",
	}
	for prefix, want := range tests {
		if got := lastIPv4(netip.MustParsePrefix(prefix)).String(); got != want {
			t.Fatalf("lastIPv4(%s) = %s, want %s", prefix, got, want)
		}
	}
}
