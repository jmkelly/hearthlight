package store

import (
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
