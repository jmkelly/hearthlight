// Package model contains the wire types shared by discovery, storage and the
// HTTP API. JSON field names are part of the browser contract.
package model

import "time"

// Service is a TCP service observed on a device.
type Service struct {
	ID        string    `json:"id"`
	Port      int       `json:"port"`
	Protocol  string    `json:"protocol"`
	Name      string    `json:"name"`
	Product   string    `json:"product,omitempty"`
	Category  string    `json:"category"`
	Source    string    `json:"source,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	// Local and Tailnet record over which network the port was observed. They
	// are sticky: a port probed successfully over either path is kept.
	Local   bool `json:"local"`
	Tailnet bool `json:"tailnet"`
	// Online is the aggregate reachability of the service.
	Online bool `json:"online"`
}

// Device is a host observed on the local network, the tailnet, or both.
// Devices are kept forever so the UI can show the history of the network,
// not just the current scan.
type Device struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	MAC       string    `json:"mac,omitempty"`
	Hostname  string    `json:"hostname,omitempty"`
	Vendor    string    `json:"vendor,omitempty"`
	Kind      string    `json:"kind"`
	Gateway   bool      `json:"gateway"`
	Self      bool      `json:"self"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	// Local and Tailnet are the dual memberships of a host. A device seen only
	// on the LAN is local, a remote Tailscale peer is tailnet, and a peer that
	// is also on the scanned LAN is both.
	Local   bool `json:"local"`
	Tailnet bool `json:"tailnet"`
	// TailnetID is the stable Tailscale node ID (ID or PublicKey) behind a
	// tailnet membership. It is omitted for LAN-only hosts.
	TailnetID     string    `json:"tailnetId,omitempty"`
	TailnetIP     string    `json:"tailnetIp,omitempty"`
	TailnetName   string    `json:"tailnetName,omitempty"`
	TailnetOS     string    `json:"tailnetOs,omitempty"`
	TailnetTags   []string  `json:"tailnetTags,omitempty"`
	TailnetOnline bool      `json:"tailnetOnline"`
	Online        bool      `json:"online"`
	Services      []Service `json:"services"`
}

// Tailnet describes the Tailscale network the host belongs to. It is omitted
// from a snapshot when the host has no usable tailnet.
type Tailnet struct {
	Name           string `json:"name,omitempty"`
	MagicDNSSuffix string `json:"magicDnsSuffix,omitempty"`
	PeersOnline    int    `json:"peersOnline"`
	PeersTotal     int    `json:"peersTotal"`
}

// TailnetPeer is one node observed in `tailscale status --json`. It is an
// observation, not stored state: the store merges it into a device record.
type TailnetPeer struct {
	// NodeID is the stable tailnet identity (Tailscale ID or PublicKey).
	NodeID   string
	HostName string
	DNSName  string
	OS       string
	IP       string
	Tags     []string
	Online   bool
	Self     bool
	LastSeen time.Time
	Kind     string
	Services []Service
}

// TailnetObservation is one reading of the local Tailscale status.
type TailnetObservation struct {
	Tailnet
	Peers []TailnetPeer
}

type Stats struct {
	DevicesOnline  int `json:"devicesOnline"`
	DevicesTotal   int `json:"devicesTotal"`
	ServicesOnline int `json:"servicesOnline"`
	ServicesTotal  int `json:"servicesTotal"`
}

// Snapshot is the complete state sent to the browser.
type Snapshot struct {
	GeneratedAt     time.Time `json:"generatedAt"`
	LastScan        time.Time `json:"lastScan"`
	Scanning        bool      `json:"scanning"`
	ScanIntervalSec int       `json:"scanIntervalSeconds"`
	ScanDurationMS  int64     `json:"scanDurationMs"`
	Networks        []string  `json:"networks"`
	LastError       string    `json:"lastError,omitempty"`
	Tailnet         *Tailnet  `json:"tailnet,omitempty"`
	Stats           Stats     `json:"stats"`
	Devices         []Device  `json:"devices"`
}
