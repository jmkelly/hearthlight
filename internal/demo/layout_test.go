package demo

import (
	"testing"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

func TestLayoutOverlaysPinnedPositionsAcrossSnapshots(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	snapshot := Generate(now)
	if len(snapshot.Devices) < 2 {
		t.Fatalf("demo snapshot has %d devices, want at least 2", len(snapshot.Devices))
	}
	id := snapshot.Devices[0].ID
	layout := NewLayout()
	if err := layout.SetDevicePosition(id, 12.5, -7.25); err != nil {
		t.Fatalf("set position: %v", err)
	}

	for _, refreshed := range []model.Snapshot{layout.Apply(snapshot), layout.Apply(Generate(now.Add(12 * time.Second)))} {
		device := findDevice(t, refreshed, id)
		if !device.Pinned || device.X != 12.5 || device.Y != -7.25 {
			t.Fatalf("position = (%v, %v) pinned=%v, want (12.5, -7.25) pinned", device.X, device.Y, device.Pinned)
		}
	}
	if snapshot.Devices[0].Pinned {
		t.Fatal("Apply mutated the snapshot it was given")
	}
}

func TestLayoutResetClearsEveryPin(t *testing.T) {
	snapshot := Generate(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	layout := NewLayout()
	for _, device := range snapshot.Devices {
		if err := layout.SetDevicePosition(device.ID, 4, 5); err != nil {
			t.Fatal(err)
		}
	}
	if err := layout.ClearDevicePositions(); err != nil {
		t.Fatalf("clear positions: %v", err)
	}
	for _, device := range layout.Apply(snapshot).Devices {
		if device.Pinned || device.X != 0 || device.Y != 0 {
			t.Fatalf("%s still pinned at (%v, %v) after reset", device.ID, device.X, device.Y)
		}
	}
}

func findDevice(t *testing.T, snapshot model.Snapshot, id string) model.Device {
	t.Helper()
	for _, device := range snapshot.Devices {
		if device.ID == id {
			return device
		}
	}
	t.Fatalf("device %s missing from snapshot", id)
	return model.Device{}
}
