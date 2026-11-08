// Package demo builds a deterministic, realistic-looking network snapshot for
// previewing the UI without scanning a real LAN.
package demo

import (
	"sort"
	"strconv"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

type serviceTemplate struct {
	port       int
	name       string
	product    string
	category   string
	online     bool
	offlineFor time.Duration
}

type deviceTemplate struct {
	id       string
	ip       string
	mac      string
	hostname string
	vendor   string
	kind     string
	gateway  bool
	online   bool
	seenFor  time.Duration
	age      time.Duration
	services []serviceTemplate
}

var templates = []deviceTemplate{
	{
		id: "mac:24:a4:3c:11:22:33", ip: "192.0.2.1", mac: "24:a4:3c:11:22:33",
		hostname: "gateway", vendor: "Ubiquiti", kind: "router", gateway: true, online: true,
		seenFor: 4 * time.Second, age: 210 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 53, name: "DNS", product: "AdGuard Home", category: "infra", online: true},
			{port: 80, name: "HTTP", product: "OpenWrt LuCI", category: "web", online: true},
			{port: 443, name: "HTTPS", product: "OpenWrt LuCI", category: "web", online: true},
		},
	},
	{
		id: "mac:b0:83:fe:44:55:66", ip: "192.0.2.30", mac: "b0:83:fe:44:55:66",
		hostname: "server", vendor: "Dell", kind: "server", online: true,
		seenFor: 9 * time.Second, age: 38 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 22, name: "SSH", product: "OpenSSH 9.6", category: "remote", online: true},
			{port: 80, name: "HTTP", product: "Caddy", category: "web", online: true},
			{port: 443, name: "HTTPS", product: "Caddy", category: "web", online: true},
			{port: 5432, name: "PostgreSQL", product: "PostgreSQL 17", category: "data", online: true},
			{port: 8090, name: "Hearthlight", product: "Hearthlight", category: "web", online: true},
			{port: 8096, name: "Jellyfin", product: "Jellyfin", category: "media", online: false, offlineFor: 3 * time.Hour},
		},
	},
	{
		id: "mac:00:11:32:77:88:99", ip: "192.0.2.20", mac: "00:11:32:77:88:99",
		hostname: "nas", vendor: "Synology", kind: "nas", online: true,
		seenFor: 12 * time.Second, age: 160 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 22, name: "SSH", product: "OpenSSH 9.6", category: "remote", online: true},
			{port: 445, name: "SMB", product: "Synology DSM", category: "data", online: true},
			{port: 5000, name: "HTTP", product: "Synology DSM", category: "web", online: true},
			{port: 5001, name: "HTTPS", product: "Synology DSM", category: "web", online: true},
		},
	},
	{
		id: "mac:dc:a6:32:aa:bb:cc", ip: "192.0.2.10", mac: "dc:a6:32:aa:bb:cc",
		hostname: "home-assistant", vendor: "Raspberry Pi", kind: "iot", online: true,
		seenFor: 6 * time.Second, age: 96 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 1883, name: "MQTT", product: "Mosquitto", category: "iot", online: true},
			{port: 8123, name: "Home Assistant", product: "Home Assistant", category: "iot", online: true},
		},
	},
	{
		id: "mac:3c:58:c2:12:34:56", ip: "192.0.2.58", mac: "3c:58:c2:12:34:56",
		hostname: "workstation", vendor: "Intel", kind: "computer", online: true,
		seenFor: 2 * time.Second, age: 320 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 22, name: "SSH", product: "OpenSSH 10.0", category: "remote", online: true},
			{port: 3000, name: "HTTP", product: "Dev server", category: "web", online: true},
			{port: 8384, name: "Syncthing", product: "Syncthing", category: "web", online: false, offlineFor: 55 * time.Minute},
		},
	},
	{
		id: "mac:b8:27:eb:12:34:56", ip: "192.0.2.5", mac: "b8:27:eb:12:34:56",
		hostname: "pi-hole", vendor: "Raspberry Pi", kind: "server", online: true,
		seenFor: 8 * time.Second, age: 280 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 53, name: "DNS", product: "Pi-hole", category: "infra", online: true},
			{port: 80, name: "HTTP", product: "Pi-hole admin", category: "web", online: true},
		},
	},
	{
		id: "mac:00:1f:29:aa:bb:cc", ip: "192.0.2.25", mac: "00:1f:29:aa:bb:cc",
		hostname: "printer", vendor: "HP", kind: "printer", online: true,
		seenFor: 31 * time.Second, age: 150 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 631, name: "IPP", product: "HP LaserJet", category: "other", online: true},
			{port: 9100, name: "RAW", product: "HP LaserJet", category: "other", online: true},
		},
	},
	{
		id: "mac:00:0e:58:11:22:33", ip: "192.0.2.42", mac: "00:0e:58:11:22:33",
		hostname: "tv", vendor: "Sony", kind: "media", online: true,
		seenFor: 45 * time.Second, age: 75 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 8009, name: "Cast", product: "Google Cast", category: "media", online: true},
		},
	},
	{
		id: "mac:3c:22:fb:aa:bb:cc", ip: "192.0.2.77", mac: "3c:22:fb:aa:bb:cc",
		hostname: "phone", vendor: "Apple", kind: "phone", online: true,
		seenFor: 18 * time.Second, age: 400 * 24 * time.Hour,
	},
	{
		id: "mac:24:0a:c4:de:ad:01", ip: "192.0.2.31", mac: "24:0a:c4:de:ad:01",
		hostname: "doorbell", vendor: "Reolink", kind: "iot", online: false,
		seenFor: 2 * 24 * time.Hour, age: 120 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 80, name: "HTTP", product: "Reolink", category: "web", online: false, offlineFor: 2 * 24 * time.Hour},
			{port: 554, name: "RTSP", product: "Reolink", category: "media", online: false, offlineFor: 2 * 24 * time.Hour},
		},
	},
	{
		id: "mac:00:14:22:aa:bb:cc", ip: "192.0.2.66", mac: "00:14:22:aa:bb:cc",
		hostname: "spare-laptop", vendor: "Dell", kind: "computer", online: false,
		seenFor: 11 * 24 * time.Hour, age: 700 * 24 * time.Hour,
		services: []serviceTemplate{
			{port: 22, name: "SSH", product: "OpenSSH 8.9", category: "remote", online: false, offlineFor: 11 * 24 * time.Hour},
		},
	},
}

// Generate rebuilds the demo snapshot relative to now so relative timestamps in
// the UI always look fresh.
func Generate(now time.Time) model.Snapshot {
	devices := make([]model.Device, 0, len(templates))
	stats := model.Stats{}

	for _, td := range templates {
		lastSeen := now.Add(-td.seenFor)
		d := model.Device{
			ID:        td.id,
			IP:        td.ip,
			MAC:       td.mac,
			Hostname:  td.hostname,
			Vendor:    td.vendor,
			Kind:      td.kind,
			Gateway:   td.gateway,
			FirstSeen: now.Add(-td.age),
			LastSeen:  lastSeen,
			Online:    td.online,
		}
		services := make([]model.Service, 0, len(td.services))
		for _, ts := range td.services {
			online := td.online && ts.online
			svcLast := now.Add(-8 * time.Second)
			if !online {
				svcLast = now.Add(-ts.offlineFor)
				if !td.online {
					svcLast = lastSeen
				}
			}
			services = append(services, model.Service{
				ID:        td.id + "/" + strconv.Itoa(ts.port),
				Port:      ts.port,
				Protocol:  "tcp",
				Name:      ts.name,
				Product:   ts.product,
				Category:  ts.category,
				Source:    "demo",
				FirstSeen: now.Add(-td.age),
				LastSeen:  svcLast,
				Online:    online,
			})
			if online {
				stats.ServicesOnline++
			}
			stats.ServicesTotal++
		}
		sort.Slice(services, func(i, j int) bool { return services[i].Port < services[j].Port })
		d.Services = services
		if d.Online {
			stats.DevicesOnline++
		}
		stats.DevicesTotal++
		devices = append(devices, d)
	}

	return model.Snapshot{
		GeneratedAt:     now,
		LastScan:        now.Add(-8 * time.Second),
		Scanning:        false,
		ScanIntervalSec: 60,
		ScanDurationMS:  684,
		Networks:        []string{"192.0.2.0/24"},
		Stats:           stats,
		Devices:         devices,
	}
}
