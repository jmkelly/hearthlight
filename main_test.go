package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
	"github.com/jmkelly/hearthlight/internal/store"
)

func TestLoadStateReopensPathAfterCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(`{"version": 2, "devices":`), 0o600); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	st := loadState(path, logger)
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	dev := model.Device{
		IP:       "192.0.2.20",
		MAC:      "aa:bb:cc:dd:ee:ff",
		Services: []model.Service{{Port: 22, Protocol: "tcp", Name: "SSH", Category: "remote"}},
	}
	if err := st.Apply([]model.Device{dev}, now, nil); err != nil {
		t.Fatalf("apply after corrupt load: %v", err)
	}

	reloaded, err := store.Load(path)
	if err != nil {
		t.Fatalf("state file was not rewritten after corrupt load: %v", err)
	}
	snap := reloaded.Snapshot(store.SnapshotOptions{Now: now, Interval: time.Minute})
	if len(snap.Devices) != 1 || snap.Devices[0].ID != "mac:aa:bb:cc:dd:ee:ff" {
		t.Fatalf("saved devices = %+v", snap.Devices)
	}

	backups, err := filepath.Glob(path + ".corrupt-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("corrupt file backups = %v, want one", backups)
	}
}
