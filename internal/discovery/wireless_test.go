package discovery

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestParseWirelessSSID(t *testing.T) {
	tests := []struct {
		name  string
		parse func(string) string
		in    string
		want  string
	}{
		{name: "iwgetid", parse: parseIwgetid, in: "HomeNet\n", want: "HomeNet"},
		{name: "iwgetid empty", parse: parseIwgetid, in: "\n", want: ""},
		{name: "nmcli active", parse: parseNmcli, in: "no:Neighbour\nyes:HomeNet\n", want: "HomeNet"},
		{name: "nmcli escaped colon", parse: parseNmcli, in: `yes:Home\:Net` + "\n", want: "Home:Net"},
		{name: "nmcli none active", parse: parseNmcli, in: "no:Neighbour\n", want: ""},
		{name: "airport network", parse: parseAirportNetwork, in: "Current Wi-Fi Network: HomeNet\n", want: "HomeNet"},
		{name: "airport not associated", parse: parseAirportNetwork, in: "You are not associated with an AirPort network.\n", want: ""},
		{name: "ipconfig summary", parse: parseSSIDField, in: "SSID : HomeNet\n", want: "HomeNet"},
		{name: "netsh ignores bssid", parse: parseSSIDField, in: "    SSID                   : HomeNet\r\n    BSSID                  : aa:bb:cc:dd:ee:ff\r\n", want: "HomeNet"},
		{name: "redacted ssid", parse: parseSSIDField, in: "    SSID                   : <redacted>\r\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.parse(tt.in); got != tt.want {
				t.Fatalf("parse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseDarwinWiFiDevices(t *testing.T) {
	out := `Hardware Port: Ethernet
Device: en1
Ethernet Address: aa:bb:cc:dd:ee:ff

Hardware Port: Wi-Fi
Device: en0
Ethernet Address: 11:22:33:44:55:66

Hardware Port: Thunderbolt Bridge
Device: bridge0
Ethernet Address: 00:11:22:33:44:55
`
	got := parseDarwinWiFiDevices(out)
	if !reflect.DeepEqual(got, []string{"en0"}) {
		t.Fatalf("parseDarwinWiFiDevices = %v, want [en0]", got)
	}
}

func TestFirstSSIDSkipsFailuresAndEmptyReadings(t *testing.T) {
	original := ssidCommandRunner
	t.Cleanup(func() { ssidCommandRunner = original })

	var ran []string
	ssidCommandRunner = func(_ context.Context, name string, _ ...string) (string, error) {
		ran = append(ran, name)
		switch name {
		case "missing":
			return "", errors.New("not found")
		case "wired":
			return "\n", nil
		case "wireless":
			return "HomeNet\n", nil
		default:
			return "LaterNet\n", nil
		}
	}

	commands := []ssidCommand{
		{name: "missing", parse: parseIwgetid},
		{name: "wired", parse: parseIwgetid},
		{name: "wireless", parse: parseIwgetid},
		{name: "later", parse: parseIwgetid},
	}
	if got := firstSSID(context.Background(), commands); got != "HomeNet" {
		t.Fatalf("firstSSID = %q, want HomeNet", got)
	}
	if !reflect.DeepEqual(ran, []string{"missing", "wired", "wireless"}) {
		t.Fatalf("commands run = %v, want to stop at the first SSID", ran)
	}
}
