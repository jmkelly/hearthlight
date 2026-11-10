package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

func sampleDevice(ip, mac string) model.Device {
	return model.Device{
		IP:        ip,
		MAC:       mac,
		Hostname:  "nas",
		Vendor:    "Synology",
		Kind:      "nas",
		FirstSeen: time.Time{},
		LastSeen:  time.Time{},
		Services: []model.Service{{
			Port:     445,
			Protocol: "tcp",
			Name:     "SMB",
			Category: "data",
		}},
	}
}

func TestApplyMergesAndRekeysWhenMACAppears(t *testing.T) {
	st, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	first := sampleDevice("192.0.2.20", "")
	if err := st.Apply([]model.Device{first}, now, []string{"192.0.2.0/24"}); err != nil {
		t.Fatal(err)
	}

	second := sampleDevice("192.0.2.20", "AA:BB:CC:DD:EE:FF")
	second.Services[0].Product = "Synology DSM"
	if err := st.Apply([]model.Device{second}, now.Add(time.Minute), nil); err != nil {
		t.Fatal(err)
	}

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(snap.Devices))
	}
	d := snap.Devices[0]
	if d.ID != "mac:aa:bb:cc:dd:ee:ff" {
		t.Fatalf("device ID = %q", d.ID)
	}
	if !d.FirstSeen.Equal(now) {
		t.Fatalf("first seen = %v, want %v", d.FirstSeen, now)
	}
	if !d.LastSeen.Equal(now.Add(time.Minute)) {
		t.Fatalf("last seen = %v, want %v", d.LastSeen, now.Add(time.Minute))
	}
	if len(d.Services) != 1 || d.Services[0].Product != "Synology DSM" {
		t.Fatalf("services not merged: %+v", d.Services)
	}
	if d.Services[0].ID != "mac:aa:bb:cc:dd:ee:ff/445" {
		t.Fatalf("service ID = %q", d.Services[0].ID)
	}
}

func TestMACIdentitySurvivesIPChange(t *testing.T) {
	st, _ := Load("")
	now := time.Now()
	if err := st.Apply([]model.Device{sampleDevice("192.0.2.20", "aa:bb:cc:dd:ee:ff")}, now, nil); err != nil {
		t.Fatal(err)
	}
	if err := st.Apply([]model.Device{sampleDevice("192.0.2.21", "aa:bb:cc:dd:ee:ff")}, now.Add(time.Minute), nil); err != nil {
		t.Fatal(err)
	}
	snap := st.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(snap.Devices))
	}
	if snap.Devices[0].IP != "192.0.2.21" {
		t.Fatalf("IP = %q, want updated address", snap.Devices[0].IP)
	}
}

func TestMACIdentitySurvivesARPMiss(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	mac := "aa:bb:cc:dd:ee:ff"
	if err := st.Apply([]model.Device{sampleDevice("192.0.2.20", mac)}, now, nil); err != nil {
		t.Fatal(err)
	}
	if err := st.Apply([]model.Device{sampleDevice("192.0.2.20", "")}, now.Add(time.Minute), nil); err != nil {
		t.Fatal(err)
	}

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(snap.Devices))
	}
	if snap.Devices[0].ID != "mac:"+mac {
		t.Fatalf("device ID = %q, want the MAC identity to survive an ARP miss", snap.Devices[0].ID)
	}
	if snap.Devices[0].MAC != mac {
		t.Fatalf("MAC = %q, want %q", snap.Devices[0].MAC, mac)
	}

	if err := st.Apply([]model.Device{sampleDevice("192.0.2.20", mac)}, now.Add(2*time.Minute), nil); err != nil {
		t.Fatal(err)
	}
	snap = st.Snapshot(SnapshotOptions{Now: now.Add(2 * time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 || snap.Devices[0].ID != "mac:"+mac {
		t.Fatalf("device flapped after the MAC returned: %+v", snap.Devices)
	}
}

func TestDHCPReuseKeepsBothDeviceHistories(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	oldDevice := sampleDevice("192.0.2.50", "aa:bb:cc:dd:ee:01")
	if err := st.Apply([]model.Device{oldDevice}, now, nil); err != nil {
		t.Fatal(err)
	}

	newDevice := sampleDevice("192.0.2.50", "aa:bb:cc:dd:ee:02")
	newDevice.Hostname = "phone"
	newDevice.Services = []model.Service{{Port: 22, Protocol: "tcp", Name: "SSH", Category: "remote"}}
	if err := st.Apply([]model.Device{newDevice}, now.Add(time.Minute), nil); err != nil {
		t.Fatal(err)
	}

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 2 {
		t.Fatalf("got %d devices, want the old and new owner of the address: %+v", len(snap.Devices), snap.Devices)
	}
	byMAC := make(map[string]model.Device, len(snap.Devices))
	for _, d := range snap.Devices {
		byMAC[d.MAC] = d
	}

	old, ok := byMAC["aa:bb:cc:dd:ee:01"]
	if !ok {
		t.Fatalf("old device disappeared: %+v", snap.Devices)
	}
	if old.ID != "mac:aa:bb:cc:dd:ee:01" || !old.FirstSeen.Equal(now) {
		t.Fatalf("old device identity = %+v, want its original ID and first-seen time", old)
	}
	if len(old.Services) != 1 || old.Services[0].Port != 445 {
		t.Fatalf("old device inherited the new device's services: %+v", old.Services)
	}

	fresh, ok := byMAC["aa:bb:cc:dd:ee:02"]
	if !ok {
		t.Fatalf("new device missing: %+v", snap.Devices)
	}
	if fresh.ID != "mac:aa:bb:cc:dd:ee:02" || !fresh.FirstSeen.Equal(now.Add(time.Minute)) {
		t.Fatalf("new device identity = %+v, want a fresh record", fresh)
	}
	if len(fresh.Services) != 1 || fresh.Services[0].Port != 22 {
		t.Fatalf("new device services = %+v", fresh.Services)
	}
}

func TestSnapshotFlipsHistoricalDevicesOffline(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if err := st.Apply([]model.Device{sampleDevice("192.0.2.20", "aa:bb:cc:dd:ee:ff")}, now, nil); err != nil {
		t.Fatal(err)
	}

	online := st.Snapshot(SnapshotOptions{Now: now.Add(2 * time.Minute), Interval: time.Minute})
	if len(online.Devices) != 1 || !online.Devices[0].Online {
		t.Fatalf("device should be online: %+v", online.Devices)
	}
	if !online.Devices[0].Services[0].Online {
		t.Fatal("service should be online")
	}
	if online.Stats.DevicesOnline != 1 || online.Stats.ServicesOnline != 1 {
		t.Fatalf("unexpected stats: %+v", online.Stats)
	}

	offline := st.Snapshot(SnapshotOptions{Now: now.Add(10 * time.Minute), Interval: time.Minute})
	if offline.Devices[0].Online {
		t.Fatal("device should be offline")
	}
	if offline.Devices[0].Services[0].Online {
		t.Fatal("service should be offline")
	}
	if offline.Stats.DevicesOnline != 0 || offline.Stats.ServicesOnline != 0 {
		t.Fatalf("unexpected stats: %+v", offline.Stats)
	}
}

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	dev := sampleDevice("192.0.2.20", "aa:bb:cc:dd:ee:ff")
	dev.Gateway = false
	if err := st.Apply([]model.Device{dev}, now, []string{"192.0.2.0/24"}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	snap := reloaded.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices after reload, want 1", len(snap.Devices))
	}
	if !snap.Devices[0].FirstSeen.Equal(now) {
		t.Fatalf("first seen not preserved: %v", snap.Devices[0].FirstSeen)
	}
	if len(snap.Networks) != 1 || snap.Networks[0] != "192.0.2.0/24" {
		t.Fatalf("networks not preserved: %v", snap.Networks)
	}
}

func TestDevicePositionPersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if err := st.Apply([]model.Device{sampleDevice("192.0.2.20", "aa:bb:cc:dd:ee:ff")}, now, nil); err != nil {
		t.Fatal(err)
	}

	const id = "mac:aa:bb:cc:dd:ee:ff"
	if err := st.SetDevicePosition(id, 12.5, -7.25); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	snap := reloaded.Snapshot(SnapshotOptions{Now: now, Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(snap.Devices))
	}
	d := snap.Devices[0]
	if !d.Pinned || d.X != 12.5 || d.Y != -7.25 {
		t.Fatalf("position = (%v, %v) pinned=%v, want (12.5, -7.25) pinned", d.X, d.Y, d.Pinned)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted struct {
		Devices map[string]struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Pinned bool    `json:"pinned"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	rec := persisted.Devices[id]
	if !rec.Pinned || rec.X != 12.5 || rec.Y != -7.25 {
		t.Fatalf("state file position = (%v, %v) pinned=%v", rec.X, rec.Y, rec.Pinned)
	}

	wire, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(wire, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"x", "y", "pinned"} {
		if _, ok := fields[name]; !ok {
			t.Fatalf("snapshot device JSON is missing %q: %s", name, wire)
		}
	}
}

func TestLoadLayoutlessStateLeavesDevicesUnpinned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	raw := `{
	  "version": 2,
	  "lastScan": "2026-11-08T12:00:00Z",
	  "devices": {
	    "ip:192.0.2.5": {
	      "id": "ip:192.0.2.5",
	      "ip": "192.0.2.5",
	      "hostname": "pi-hole",
	      "lastSeen": "2026-11-08T12:00:00Z",
	      "local": true
	    }
	  }
	}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	snap := st.Snapshot(SnapshotOptions{Now: time.Date(2026, 11, 8, 12, 1, 0, 0, time.UTC), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(snap.Devices))
	}
	d := snap.Devices[0]
	if d.Pinned || d.X != 0 || d.Y != 0 {
		t.Fatalf("layoutless state loaded position = (%v, %v) pinned=%v, want unpinned", d.X, d.Y, d.Pinned)
	}
}

func TestSetDevicePositionUnknownIDChangesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if err := st.Apply([]model.Device{sampleDevice("192.0.2.20", "aa:bb:cc:dd:ee:ff")}, now, nil); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := st.SetDevicePosition("mac:aa:bb:cc:dd:ee:99", 4, 5); err == nil {
		t.Fatal("SetDevicePosition with an unknown id returned no error")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("state file changed after an unknown device id")
	}
	snap := st.Snapshot(SnapshotOptions{Now: now, Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(snap.Devices))
	}
	if d := snap.Devices[0]; d.Pinned || d.X != 0 || d.Y != 0 {
		t.Fatalf("device changed after an unknown id: position (%v, %v) pinned=%v", d.X, d.Y, d.Pinned)
	}
}

func TestClearDevicePositionsUnpinsAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	devices := []model.Device{
		sampleDevice("192.0.2.20", "aa:bb:cc:dd:ee:01"),
		sampleDevice("192.0.2.21", "aa:bb:cc:dd:ee:02"),
	}
	if err := st.Apply(devices, now, nil); err != nil {
		t.Fatal(err)
	}
	for i, d := range devices {
		if err := st.SetDevicePosition("mac:"+d.MAC, float64(i+1), float64(i+2)); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.ClearDevicePositions(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	snap := reloaded.Snapshot(SnapshotOptions{Now: now, Interval: time.Minute})
	if len(snap.Devices) != 2 {
		t.Fatalf("got %d devices, want 2", len(snap.Devices))
	}
	for _, d := range snap.Devices {
		if d.Pinned || d.X != 0 || d.Y != 0 {
			t.Fatalf("device %s still has position (%v, %v) pinned=%v", d.ID, d.X, d.Y, d.Pinned)
		}
	}
}

func TestLoadMissingStateIsEmpty(t *testing.T) {
	st, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	snap := st.Snapshot(SnapshotOptions{Now: time.Now(), Interval: time.Minute})
	if len(snap.Devices) != 0 {
		t.Fatalf("got %d devices, want 0", len(snap.Devices))
	}
	if snap.Devices == nil {
		t.Fatal("devices must serialize as an empty array")
	}
}

func tailnetPeer(id, hostname, ip string, online bool) model.TailnetPeer {
	return model.TailnetPeer{
		NodeID:   id,
		HostName: hostname,
		DNSName:  hostname + ".foxglove-heron.ts.net.",
		OS:       "linux",
		IP:       ip,
		Online:   online,
	}
}

func applyTailnet(t *testing.T, st *Store, now time.Time, peers ...model.TailnetPeer) {
	t.Helper()
	err := st.ApplyTailnet(model.TailnetObservation{
		Tailnet: model.Tailnet{Name: "foxglove-heron", MagicDNSSuffix: "foxglove-heron.ts.net"},
		Peers:   peers,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDualMembershipMergesByHostname(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	lan := sampleDevice("192.0.2.30", "aa:bb:cc:dd:ee:ff")
	lan.Hostname = "server"
	lan.Services = []model.Service{
		{Port: 22, Protocol: "tcp", Name: "SSH", Category: "remote"},
		{Port: 5432, Protocol: "tcp", Name: "PostgreSQL", Category: "data"},
	}
	if err := st.Apply([]model.Device{lan}, now, nil); err != nil {
		t.Fatal(err)
	}

	peer := tailnetPeer("nServer1", "server", "100.101.102.42", true)
	peer.Services = []model.Service{
		{Port: 22, Protocol: "tcp", Name: "SSH", Category: "remote"},
		{Port: 8090, Protocol: "tcp", Name: "Hearthlight", Category: "web"},
	}
	applyTailnet(t, st, now.Add(2*time.Second), peer)

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(2 * time.Second), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want one dual-membership device: %+v", len(snap.Devices), snap.Devices)
	}
	d := snap.Devices[0]
	if !d.Local || !d.Tailnet {
		t.Fatalf("membership = local:%v tailnet:%v, want both", d.Local, d.Tailnet)
	}
	if d.ID != "mac:aa:bb:cc:dd:ee:ff" || d.IP != "192.0.2.30" {
		t.Fatalf("LAN identity not preserved: %+v", d)
	}
	if d.TailnetIP != "100.101.102.42" || d.TailnetID != "nServer1" {
		t.Fatalf("tailnet metadata = %+v", d)
	}
	if !d.TailnetOnline || !d.Online {
		t.Fatalf("device should be online: %+v", d)
	}
	if len(d.Services) != 3 {
		t.Fatalf("services = %+v, want 3 merged by port", d.Services)
	}
	for _, svc := range d.Services {
		switch svc.Port {
		case 22:
			if !svc.Local || !svc.Tailnet {
				t.Fatalf("port 22 reach = local:%v tailnet:%v, want both", svc.Local, svc.Tailnet)
			}
		case 5432:
			if !svc.Local || svc.Tailnet {
				t.Fatalf("port 5432 reach = local:%v tailnet:%v, want LAN only", svc.Local, svc.Tailnet)
			}
		case 8090:
			if svc.Local || !svc.Tailnet {
				t.Fatalf("port 8090 reach = local:%v tailnet:%v, want tailnet only", svc.Local, svc.Tailnet)
			}
		}
	}
	if snap.Tailnet == nil || snap.Tailnet.Name != "foxglove-heron" || snap.Tailnet.PeersTotal != 1 {
		t.Fatalf("tailnet snapshot = %+v", snap.Tailnet)
	}
}

func TestTailnetOnlyPeerStaysOnlineWithoutServices(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	applyTailnet(t, st, now, tailnetPeer("nRemote1", "cloud-1", "100.101.102.9", true))

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices", len(snap.Devices))
	}
	d := snap.Devices[0]
	if d.ID != "ts:nRemote1" {
		t.Fatalf("tailnet-only ID = %q, want ts:<node id>", d.ID)
	}
	if d.Local || !d.Tailnet {
		t.Fatalf("membership = local:%v tailnet:%v, want tailnet only", d.Local, d.Tailnet)
	}
	if !d.Online || !d.TailnetOnline {
		t.Fatalf("an online peer must stay online even without services: %+v", d)
	}
	if snap.Stats.DevicesOnline != 1 || snap.Tailnet.PeersOnline != 1 {
		t.Fatalf("stats = %+v tailnet = %+v", snap.Stats, snap.Tailnet)
	}
}

func TestOfflinePeerKeepsTailscaleLastSeen(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	peer := tailnetPeer("nTablet1", "ipad", "100.101.102.80", false)
	peer.LastSeen = now.Add(-2 * time.Hour)
	applyTailnet(t, st, now, peer)

	snap := st.Snapshot(SnapshotOptions{Now: now, Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices", len(snap.Devices))
	}
	d := snap.Devices[0]
	if d.Online || d.TailnetOnline {
		t.Fatalf("offline peer marked online: %+v", d)
	}
	if !d.LastSeen.Equal(now.Add(-2 * time.Hour)) {
		t.Fatalf("last seen = %v, want the Tailscale timestamp", d.LastSeen)
	}
}

func TestTailnetOnlyRecordMergesIntoLaterLANRecord(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	peer := tailnetPeer("nNas1", "nas", "100.101.102.18", true)
	peer.Services = []model.Service{{Port: 445, Protocol: "tcp", Name: "SMB", Category: "data"}}
	applyTailnet(t, st, now, peer)

	lan := sampleDevice("192.0.2.20", "aa:bb:cc:dd:ee:ff")
	lan.Hostname = "nas"
	lan.Services = append(lan.Services, model.Service{Port: 22, Protocol: "tcp", Name: "SSH", Category: "remote"})
	if err := st.Apply([]model.Device{lan}, now.Add(time.Minute), nil); err != nil {
		t.Fatal(err)
	}
	applyTailnet(t, st, now.Add(time.Minute), peer)

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(2 * time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices, want the tailnet record folded into the LAN record", len(snap.Devices))
	}
	d := snap.Devices[0]
	if d.ID != "mac:aa:bb:cc:dd:ee:ff" {
		t.Fatalf("ID = %q, want LAN identity", d.ID)
	}
	if !d.FirstSeen.Equal(now) {
		t.Fatalf("FirstSeen = %v, want earliest observation %v", d.FirstSeen, now)
	}
	ports := make(map[int]bool)
	for _, svc := range d.Services {
		ports[svc.Port] = true
		if svc.Port == 445 && !svc.Tailnet {
			t.Fatalf("tailnet service lost during merge: %+v", svc)
		}
	}
	if !ports[445] || !ports[22] {
		t.Fatalf("services after merge = %+v", d.Services)
	}
}

func TestAmbiguousHostnameIsNotMerged(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	a := sampleDevice("192.0.2.30", "aa:bb:cc:dd:ee:01")
	a.Hostname = "server"
	b := sampleDevice("192.0.2.31", "aa:bb:cc:dd:ee:02")
	b.Hostname = "server"
	if err := st.Apply([]model.Device{a, b}, now, nil); err != nil {
		t.Fatal(err)
	}
	applyTailnet(t, st, now, tailnetPeer("nServer1", "server", "100.101.102.42", true))

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 3 {
		t.Fatalf("got %d devices, want two LAN records plus one tailnet-only record", len(snap.Devices))
	}
	for _, d := range snap.Devices {
		if d.ID == "ts:nServer1" {
			if d.Local {
				t.Fatal("ambiguous peer must not be marked local")
			}
			return
		}
	}
	t.Fatal("tailnet record missing")
}

func TestSelfPeerJoinsLANSelf(t *testing.T) {
	st, _ := Load("")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	self := sampleDevice("192.0.2.58", "aa:bb:cc:dd:ee:ff")
	self.Hostname = "laptop"
	self.Self = true
	if err := st.Apply([]model.Device{self}, now, nil); err != nil {
		t.Fatal(err)
	}
	peer := tailnetPeer("nSelf1", "renamed-by-tailscale", "100.101.102.103", true)
	peer.Self = true
	applyTailnet(t, st, now, peer)

	snap := st.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if len(snap.Devices) != 1 || !snap.Devices[0].Tailnet || !snap.Devices[0].Local {
		t.Fatalf("self did not merge by flag: %+v", snap.Devices)
	}
}

func TestLoadMigratesVersionOneStateToLocal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	raw := `{
	  "version": 1,
	  "lastScan": "2026-11-08T12:00:00Z",
	  "networks": ["192.0.2.0/24"],
	  "devices": {
	    "ip:192.0.2.5": {
	      "id": "ip:192.0.2.5",
	      "ip": "192.0.2.5",
	      "hostname": "pi-hole",
	      "lastSeen": "2026-11-08T12:00:00Z",
	      "services": {
	        "80": {"port": 80, "protocol": "tcp", "name": "HTTP", "category": "web", "lastSeen": "2026-11-08T12:00:00Z"}
	      }
	    }
	  }
	}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 1, 0, 0, time.UTC)
	snap := st.Snapshot(SnapshotOptions{Now: now, Interval: time.Minute})
	if len(snap.Devices) != 1 {
		t.Fatalf("got %d devices", len(snap.Devices))
	}
	d := snap.Devices[0]
	if !d.Local || d.Tailnet {
		t.Fatalf("migrated membership = local:%v tailnet:%v, want local only", d.Local, d.Tailnet)
	}
	if len(d.Services) != 1 || !d.Services[0].Local || d.Services[0].Tailnet {
		t.Fatalf("migrated service reach = %+v", d.Services)
	}
	if snap.Tailnet != nil {
		t.Fatalf("tailnet must be omitted without data: %+v", snap.Tailnet)
	}
}

func TestTailnetNamePersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, _ := Load(path)
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	applyTailnet(t, st, now, tailnetPeer("nRemote1", "cloud-1", "100.101.102.9", true))

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	snap := reloaded.Snapshot(SnapshotOptions{Now: now.Add(time.Minute), Interval: time.Minute})
	if snap.Tailnet == nil || snap.Tailnet.MagicDNSSuffix != "foxglove-heron.ts.net" {
		t.Fatalf("tailnet metadata lost on reload: %+v", snap.Tailnet)
	}
	if len(snap.Devices) != 1 || !snap.Devices[0].Tailnet {
		t.Fatalf("tailnet device lost on reload: %+v", snap.Devices)
	}
}
