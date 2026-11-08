// Package store persists observations and merges repeated scans into a single
// inventory of devices and services. It never deletes history automatically:
// devices that disappear become offline, not forgotten.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jmkelly/hearthlight/internal/model"
)

const stateVersion = 1

// Store is safe for concurrent use.
type Store struct {
	mu       sync.Mutex
	path     string
	devices  map[string]*deviceRecord
	lastScan time.Time
	networks []string
}

type persisted struct {
	Version  int                      `json:"version"`
	LastScan time.Time                `json:"lastScan"`
	Networks []string                 `json:"networks,omitempty"`
	Devices  map[string]*deviceRecord `json:"devices"`
}

type deviceRecord struct {
	ID        string                 `json:"id"`
	IP        string                 `json:"ip"`
	MAC       string                 `json:"mac,omitempty"`
	Hostname  string                 `json:"hostname,omitempty"`
	Vendor    string                 `json:"vendor,omitempty"`
	Kind      string                 `json:"kind,omitempty"`
	Gateway   bool                   `json:"gateway,omitempty"`
	Self      bool                   `json:"self,omitempty"`
	FirstSeen time.Time              `json:"firstSeen"`
	LastSeen  time.Time              `json:"lastSeen"`
	Services  map[int]*serviceRecord `json:"services,omitempty"`
}

type serviceRecord struct {
	Port      int       `json:"port"`
	Protocol  string    `json:"protocol"`
	Name      string    `json:"name"`
	Product   string    `json:"product,omitempty"`
	Category  string    `json:"category"`
	Source    string    `json:"source,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

// Load reads a state file. A missing file yields an empty store, not an error.
// Pass an empty path for an in-memory store.
func Load(path string) (*Store, error) {
	s := &Store{
		path:    path,
		devices: make(map[string]*deviceRecord),
	}
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	if p.Devices != nil {
		s.devices = p.Devices
	}
	s.lastScan = p.LastScan
	s.networks = append([]string(nil), p.Networks...)
	for key, rec := range s.devices {
		if rec.Services == nil {
			rec.Services = make(map[int]*serviceRecord)
		}
		if rec.ID == "" {
			rec.ID = key
		}
	}
	return s, nil
}

// Apply merges one scan into the store and writes the state file.
func (s *Store) Apply(observed []model.Device, now time.Time, networks []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range observed {
		obs := observed[i]
		mac := normalizeMAC(obs.MAC)
		recordKey, rec := s.findLocked(mac, obs.IP)
		if rec == nil {
			rec = &deviceRecord{
				FirstSeen: now,
				Services:  make(map[int]*serviceRecord),
			}
		}

		newID := deviceID(obs.IP, mac)
		oldID := rec.ID
		if oldID != newID {
			if oldID != "" {
				delete(s.devices, oldID)
			}
			if recordKey != "" && recordKey != oldID {
				delete(s.devices, recordKey)
			}
			rec.ID = newID
		}
		s.devices[rec.ID] = rec

		rec.IP = obs.IP
		if mac != "" {
			rec.MAC = mac
		}
		if obs.Hostname != "" {
			rec.Hostname = obs.Hostname
		}
		if obs.Vendor != "" {
			rec.Vendor = obs.Vendor
		}
		if obs.Kind != "" {
			rec.Kind = obs.Kind
		}
		rec.Gateway = obs.Gateway
		rec.Self = obs.Self
		if rec.FirstSeen.IsZero() || (!obs.FirstSeen.IsZero() && obs.FirstSeen.Before(rec.FirstSeen)) {
			rec.FirstSeen = obs.FirstSeen
		}
		rec.LastSeen = now

		for _, svc := range obs.Services {
			sr := rec.Services[svc.Port]
			if sr == nil {
				sr = &serviceRecord{
					Port:      svc.Port,
					FirstSeen: now,
				}
				rec.Services[svc.Port] = sr
			}
			if !svc.FirstSeen.IsZero() && (sr.FirstSeen.IsZero() || svc.FirstSeen.Before(sr.FirstSeen)) {
				sr.FirstSeen = svc.FirstSeen
			}
			if svc.Protocol != "" {
				sr.Protocol = svc.Protocol
			}
			if svc.Name != "" {
				sr.Name = svc.Name
			}
			if svc.Product != "" {
				sr.Product = svc.Product
			}
			if svc.Category != "" {
				sr.Category = svc.Category
			}
			if svc.Source != "" {
				sr.Source = svc.Source
			}
			sr.LastSeen = now
		}
	}

	s.lastScan = now
	s.networks = cleanNetworks(networks)
	return s.saveLocked()
}

func (s *Store) findLocked(mac, ip string) (string, *deviceRecord) {
	if mac != "" {
		for key, rec := range s.devices {
			if rec.MAC == mac {
				return key, rec
			}
		}
	}
	if ip != "" {
		for key, rec := range s.devices {
			if rec.IP == ip {
				return key, rec
			}
		}
	}
	return "", nil
}

// SnapshotOptions controls the derived online/offline view of the store.
type SnapshotOptions struct {
	Now          time.Time
	Interval     time.Duration
	Scanning     bool
	ScanDuration time.Duration
	LastError    string
}

// Snapshot returns a deep copy of the inventory with current status flags.
func (s *Store) Snapshot(opts SnapshotOptions) model.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	onlineFor := 3 * opts.Interval
	if onlineFor < 45*time.Second {
		onlineFor = 45 * time.Second
	}

	devices := make([]model.Device, 0, len(s.devices))
	stats := model.Stats{}
	for _, rec := range s.devices {
		d := model.Device{
			ID:        rec.ID,
			IP:        rec.IP,
			MAC:       rec.MAC,
			Hostname:  rec.Hostname,
			Vendor:    rec.Vendor,
			Kind:      rec.Kind,
			Gateway:   rec.Gateway,
			Self:      rec.Self,
			FirstSeen: rec.FirstSeen,
			LastSeen:  rec.LastSeen,
		}
		if d.Kind == "" {
			d.Kind = "device"
		}
		d.Online = !rec.LastSeen.IsZero() && now.Sub(rec.LastSeen) <= onlineFor

		services := make([]model.Service, 0, len(rec.Services))
		for _, sr := range rec.Services {
			svc := model.Service{
				ID:        fmt.Sprintf("%s/%d", d.ID, sr.Port),
				Port:      sr.Port,
				Protocol:  sr.Protocol,
				Name:      sr.Name,
				Product:   sr.Product,
				Category:  sr.Category,
				Source:    sr.Source,
				FirstSeen: sr.FirstSeen,
				LastSeen:  sr.LastSeen,
			}
			if svc.Protocol == "" {
				svc.Protocol = "tcp"
			}
			if svc.Category == "" {
				svc.Category = "other"
			}
			svc.Online = d.Online && !sr.LastSeen.IsZero() && now.Sub(sr.LastSeen) <= onlineFor
			if svc.Online {
				stats.ServicesOnline++
			}
			stats.ServicesTotal++
			services = append(services, svc)
		}
		sort.Slice(services, func(i, j int) bool { return services[i].Port < services[j].Port })
		d.Services = services
		if d.Online {
			stats.DevicesOnline++
		}
		stats.DevicesTotal++
		devices = append(devices, d)
	}
	sort.Slice(devices, func(i, j int) bool { return ipLess(devices[i].IP, devices[j].IP) })

	return model.Snapshot{
		GeneratedAt:     now,
		LastScan:        s.lastScan,
		Scanning:        opts.Scanning,
		ScanIntervalSec: int(opts.Interval / time.Second),
		ScanDurationMS:  opts.ScanDuration.Milliseconds(),
		Networks:        append([]string(nil), s.networks...),
		LastError:       opts.LastError,
		Stats:           stats,
		Devices:         devices,
	}
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	p := persisted{
		Version:  stateVersion,
		LastScan: s.lastScan,
		Networks: s.networks,
		Devices:  s.devices,
	}
	tmp, err := os.CreateTemp(dir, ".hearthlight-*.json")
	if err != nil {
		return fmt.Errorf("create state temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close state: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod state: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	cleanup = false
	return nil
}

func deviceID(ip, mac string) string {
	if mac != "" {
		return "mac:" + mac
	}
	return "ip:" + ip
}

func normalizeMAC(raw string) string {
	mac := strings.ToLower(strings.TrimSpace(raw))
	if len(mac) != 17 || mac == "00:00:00:00:00:00" {
		return ""
	}
	for i, r := range mac {
		if i%3 == 2 {
			if r != ':' {
				return ""
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return ""
		}
	}
	return mac
}

func cleanNetworks(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, n := range in {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func ipLess(a, b string) bool {
	aa, errA := netip.ParseAddr(a)
	bb, errB := netip.ParseAddr(b)
	switch {
	case errA == nil && errB == nil:
		return aa.Less(bb)
	case errA == nil:
		return true
	case errB == nil:
		return false
	default:
		return a < b
	}
}
