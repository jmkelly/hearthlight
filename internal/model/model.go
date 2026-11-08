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
	Online    bool      `json:"online"`
}

// Device is a host observed on the local network. Devices are kept forever so
// the UI can show the history of the network, not just the current scan.
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
	Online    bool      `json:"online"`
	Services  []Service `json:"services"`
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
	Stats           Stats     `json:"stats"`
	Devices         []Device  `json:"devices"`
}
