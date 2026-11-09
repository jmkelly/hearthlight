package discovery

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// wirelessSSIDTimeout bounds the whole SSID lookup so a wedged OS helper never
// stalls a LAN scan. A wired host or missing tool simply yields no SSID.
const wirelessSSIDTimeout = 2 * time.Second

type ssidCommand struct {
	name  string
	args  []string
	parse func(string) string
}

// ssidCommandRunner is a test seam for the OS command sequence.
var ssidCommandRunner = runSSIDCommand

func runSSIDCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// readWirelessSSID returns the current Wi-Fi network name, or an empty string
// when the host is wired, the helper is missing, or the OS withholds the name.
func readWirelessSSID(ctx context.Context) string {
	lookupCtx, cancel := context.WithTimeout(ctx, wirelessSSIDTimeout)
	defer cancel()
	return firstSSID(lookupCtx, wirelessSSIDCommands(lookupCtx))
}

func firstSSID(ctx context.Context, commands []ssidCommand) string {
	for _, command := range commands {
		out, err := ssidCommandRunner(ctx, command.name, command.args...)
		if err != nil {
			continue
		}
		if ssid := command.parse(out); ssid != "" {
			return ssid
		}
	}
	return ""
}

func wirelessSSIDCommands(ctx context.Context) []ssidCommand {
	switch runtime.GOOS {
	case "linux":
		return []ssidCommand{
			{name: "iwgetid", args: []string{"-r"}, parse: parseIwgetid},
			{name: "nmcli", args: []string{"-t", "-f", "active,ssid", "dev", "wifi"}, parse: parseNmcli},
		}
	case "darwin":
		devices := darwinWiFiDevices(ctx)
		commands := make([]ssidCommand, 0, len(devices)*2)
		for _, device := range devices {
			commands = append(commands, ssidCommand{
				name:  toolPath("networksetup", "/usr/sbin/networksetup"),
				args:  []string{"-getairportnetwork", device},
				parse: parseAirportNetwork,
			})
		}
		for _, device := range devices {
			commands = append(commands, ssidCommand{
				name:  toolPath("ipconfig", "/usr/sbin/ipconfig"),
				args:  []string{"getsummary", device},
				parse: parseSSIDField,
			})
		}
		return commands
	case "windows":
		return []ssidCommand{{
			name:  toolPath("netsh", filepath.Join(os.Getenv("SystemRoot"), "System32", "netsh.exe")),
			args:  []string{"wlan", "show", "interfaces"},
			parse: parseSSIDField,
		}}
	default:
		return nil
	}
}

func darwinWiFiDevices(ctx context.Context) []string {
	out, err := ssidCommandRunner(ctx, toolPath("networksetup", "/usr/sbin/networksetup"), "-listallhardwareports")
	if err != nil {
		return []string{"en0"}
	}
	devices := parseDarwinWiFiDevices(out)
	if len(devices) == 0 {
		return []string{"en0"}
	}
	return devices
}

// toolPath prefers PATH but falls back to the system location so a binary
// launched from Finder or a service with a minimal PATH still finds the tool.
func toolPath(name string, fallbacks ...string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	for _, fallback := range fallbacks {
		if info, err := os.Stat(fallback); err == nil && !info.IsDir() {
			return fallback
		}
	}
	return name
}

func parseIwgetid(out string) string {
	return cleanSSID(out)
}

// parseNmcli reads terse `nmcli` output. An active line is "yes:<ssid>" and
// colons inside the SSID are backslash-escaped.
func parseNmcli(out string) string {
	for _, line := range strings.Split(out, "\n") {
		active, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(active), "yes") {
			continue
		}
		return cleanSSID(unescapeNmcli(value))
	}
	return ""
}

func unescapeNmcli(value string) string {
	var out strings.Builder
	escaped := false
	for _, character := range value {
		if escaped {
			out.WriteRune(character)
			escaped = false
			continue
		}
		if character == '\\' {
			escaped = true
			continue
		}
		out.WriteRune(character)
	}
	return out.String()
}

func parseAirportNetwork(out string) string {
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "Current Wi-Fi Network", "Current AirPort Network":
			return cleanSSID(value)
		}
	}
	return ""
}

// parseSSIDField reads the "SSID : <name>" line from `ipconfig getsummary` and
// `netsh wlan show interfaces`. Matching the whole key avoids confusing it
// with the adjacent BSSID line.
func parseSSIDField(out string) string {
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "SSID") {
			continue
		}
		return cleanSSID(value)
	}
	return ""
}

// parseDarwinWiFiDevices extracts the Wi-Fi device names from
// `networksetup -listallhardwareports`.
func parseDarwinWiFiDevices(out string) []string {
	var devices []string
	wireless := false
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			wireless = false
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "Hardware Port":
			port := strings.TrimSpace(value)
			wireless = port == "Wi-Fi" || port == "AirPort"
		case "Device":
			if wireless {
				if device := strings.TrimSpace(value); device != "" {
					devices = append(devices, device)
				}
				wireless = false
			}
		}
	}
	return devices
}

func cleanSSID(raw string) string {
	ssid := strings.TrimSpace(raw)
	if strings.EqualFold(ssid, "<redacted>") {
		return ""
	}
	return ssid
}
