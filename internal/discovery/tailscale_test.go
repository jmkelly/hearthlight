package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleStatus = `{
  "BackendState": "Running",
  "MagicDNSSuffix": "foxglove-heron.ts.net",
  "CurrentTailnet": {"Name": "foxglove-heron", "MagicDNSSuffix": "foxglove-heron.ts.net", "MagicDNSEnabled": true},
  "Self": {
    "ID": "nSelf01",
    "PublicKey": "nodekey:self",
    "HostName": "workstation",
    "DNSName": "workstation.foxglove-heron.ts.net.",
    "OS": "linux",
    "TailscaleIPs": ["fd7a:115c:a1e0::1", "100.101.102.103"],
    "Online": false,
    "Tags": ["tag:dev"]
  },
  "Peer": {
    "nodekey:abc": {
      "ID": "",
      "PublicKey": "nodekey:abc",
      "HostName": "macbook-air",
      "DNSName": "macbook-air.foxglove-heron.ts.net.",
      "OS": "macOS",
      "TailscaleIPs": ["100.101.102.64", "fd7a:115c:a1e0::64"],
      "Online": true,
      "LastSeen": "2026-11-08T06:30:00.1Z"
    },
    "nOffline": {
      "ID": "nOffline",
      "PublicKey": "nodekey:off",
      "HostName": "ipad",
      "DNSName": "ipad.foxglove-heron.ts.net.",
      "OS": "iOS",
      "TailscaleIPs": ["100.101.102.80"],
      "Online": false,
      "LastSeen": "2026-11-08T04:00:00Z"
    }
  }
}`

func TestParseTailnetStatus(t *testing.T) {
	obs, err := parseTailnetStatus([]byte(sampleStatus))
	if err != nil {
		t.Fatal(err)
	}
	if obs.Name != "foxglove-heron" || obs.MagicDNSSuffix != "foxglove-heron.ts.net" {
		t.Fatalf("tailnet metadata = %+v", obs.Tailnet)
	}
	if len(obs.Peers) != 3 {
		t.Fatalf("got %d peers, want self plus two peers", len(obs.Peers))
	}
	self := obs.Peers[0]
	if !self.Self || !self.Online {
		t.Fatalf("self peer = %+v, want self and online", self)
	}
	if self.IP != "100.101.102.103" {
		t.Fatalf("self IP = %q, want the CGNAT IPv4", self.IP)
	}
	if self.NodeID != "nSelf01" {
		t.Fatalf("self node ID = %q", self.NodeID)
	}
	if len(self.Tags) != 1 || self.Tags[0] != "tag:dev" {
		t.Fatalf("tags = %v", self.Tags)
	}

	byName := make(map[string]int)
	for _, peer := range obs.Peers {
		byName[peer.HostName] = 1
	}
	if byName["macbook-air"] != 1 || byName["ipad"] != 1 {
		t.Fatalf("peers missing: %+v", obs.Peers)
	}
	for _, peer := range obs.Peers {
		if peer.HostName != "macbook-air" {
			continue
		}
		if peer.NodeID != "nodekey:abc" {
			t.Fatalf("ID fallback = %q, want PublicKey", peer.NodeID)
		}
		if peer.Kind != "computer" {
			t.Fatalf("kind = %q, want computer for macOS", peer.Kind)
		}
		if peer.LastSeen.IsZero() {
			t.Fatal("LastSeen not parsed")
		}
	}
}

func TestParseTailnetStatusRejectsStoppedBackend(t *testing.T) {
	_, err := parseTailnetStatus([]byte(`{"BackendState":"Stopped","Self":null}`))
	if err == nil || !strings.Contains(err.Error(), "Stopped") {
		t.Fatalf("error = %v, want a stopped-backend error", err)
	}
}

func TestParseTailnetStatusRejectsMalformedJSON(t *testing.T) {
	if _, err := parseTailnetStatus([]byte(`{"Self":`)); err == nil {
		t.Fatal("malformed JSON must return an error")
	}
	if _, err := parseTailnetStatus([]byte(`{"BackendState":"Running","Peer":{}}`)); err == nil {
		t.Fatal("a status without self must return an error")
	}
}

func TestTailscaleBinaryMissingIsGraceful(t *testing.T) {
	t.Setenv("HEARTHLIGHT_TAILSCALE", "/definitely/not/here")
	if _, err := tailscaleBinary(); err == nil {
		t.Fatal("expected an error for a bad HEARTHLIGHT_TAILSCALE override")
	}
}

func TestTailscaleBinaryOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tailscale")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tailscaleBinaryOverride = path
	t.Cleanup(func() { tailscaleBinaryOverride = "" })
	got, err := tailscaleBinary()
	if err != nil || got != path {
		t.Fatalf("tailscaleBinary() = %q, %v", got, err)
	}
}
