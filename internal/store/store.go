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

// stateVersion 2 adds dual local/tailnet membership. Version 1 records could
// only have been observed on the LAN, so they migrate to local=true.
const stateVersion = 2

// Store is safe for concurrent use.
type Store struct {
	mu       sync.Mutex
	path     string
	devices  map[string]*deviceRecord
	lastScan time.Time
	networks []string
	tailnet  model.Tailnet
}

type persisted struct {
	Version  int                      `json:"version"`
	LastScan time.Time                `json:"lastScan"`
	Networks []string                 `json:"networks,omitempty"`
	Tailnet  *model.Tailnet           `json:"tailnet,omitempty"`
	Devices  map[string]*deviceRecord `json:"devices"`
}

type deviceRecord struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	MAC       string    `json:"mac,omitempty"`
	Hostname  string    `json:"hostname,omitempty"`
	Vendor    string    `json:"vendor,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	Gateway   bool      `json:"gateway,omitempty"`
	Self      bool      `json:"self,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	Local     bool      `json:"local,omitempty"`
	Tailnet   bool      `json:"tailnet,omitempty"`

	TailnetID       string    `json:"tailnetId,omitempty"`
	TailnetIP       string    `json:"tailnetIp,omitempty"`
	TailnetName     string    `json:"tailnetName,omitempty"`
	TailnetOS       string    `json:"tailnetOs,omitempty"`
	TailnetTags     []string  `json:"tailnetTags,omitempty"`
	LocalLastSeen   time.Time `json:"localLastSeen,omitempty"`
	TailnetLastSeen time.Time `json:"tailnetLastSeen,omitempty"`

	Services map[int]*serviceRecord `json:"services,omitempty"`
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
	Local     bool      `json:"local,omitempty"`
	Tailnet   bool      `json:"tailnet,omitempty"`
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
	if p.Tailnet != nil {
		s.tailnet = *p.Tailnet
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
		if p.Version < 2 {
			// A pre-tailnet state file only contains LAN observations.
			rec.Local = true
			if rec.LocalLastSeen.IsZero() {
				rec.LocalLastSeen = rec.LastSeen
			}
			for _, svc := range rec.Services {
				svc.Local = true
			}
		}
	}
	return s, nil
}

// Apply merges one LAN scan into the store and writes the state file.
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
		if mac == "" && rec.MAC != "" {
			// An ARP miss must not rename a MAC-keyed device to an IP-keyed one.
			// Keep the stable identity until the MAC is observed again.
			newID = deviceID(obs.IP, rec.MAC)
		}
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
		rec.Local = true
		rec.LocalLastSeen = now
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
			sr.Local = true
			sr.LastSeen = now
		}
	}

	s.lastScan = now
	s.networks = cleanNetworks(networks)
	return s.saveLocked()
}

// ApplyTailnet merges one Tailscale status reading. Peers that share a unique
// normalized short hostname with a LAN device are folded into that record;
// the others become tailnet-only devices keyed by ts:<node id>. An ambiguous
// hostname is never merged.
func (s *Store) ApplyTailnet(obs model.TailnetObservation, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	online := 0
	for _, peer := range obs.Peers {
		if peer.Online {
			online++
		}
	}
	// The snapshot counts come from the latest successful reading, not from
	// every peer ever seen, so decommissioned peers do not inflate "x/y up".
	s.tailnet = model.Tailnet{
		Name:           obs.Name,
		MagicDNSSuffix: obs.MagicDNSSuffix,
		PeersOnline:    online,
		PeersTotal:     len(obs.Peers),
	}

	nameCounts := make(map[string]int)
	for _, peer := range obs.Peers {
		for _, name := range peerNames(peer) {
			nameCounts[name]++
		}
	}
	lanIndex := s.lanNameIndexLocked()

	for _, peer := range obs.Peers {
		if peer.NodeID == "" && peer.IP == "" {
			continue
		}
		named := s.matchLANLocked(peer, nameCounts, lanIndex)
		bound := s.findByTailnetIDLocked(peer.NodeID)

		if named != nil && bound != nil && named != bound {
			// The peer previously existed as tailnet-only and has now turned up
			// on the scanned LAN. Fold the old record into the LAN record.
			s.mergeRecordLocked(named, bound)
			lanIndex = s.lanNameIndexLocked()
		}

		rec := named
		if rec == nil {
			rec = bound
		}
		if rec == nil {
			rec = s.newTailnetRecordLocked(peer)
		}
		s.applyPeerLocked(rec, peer, now)
	}

	return s.saveLocked()
}

func (s *Store) newTailnetRecordLocked(peer model.TailnetPeer) *deviceRecord {
	key := tailnetKey(peer)
	rec := s.devices[key]
	if rec == nil {
		rec = &deviceRecord{
			ID:       key,
			Services: make(map[int]*serviceRecord),
		}
		s.devices[key] = rec
	}
	return rec
}

func (s *Store) applyPeerLocked(rec *deviceRecord, peer model.TailnetPeer, now time.Time) {
	if rec.Services == nil {
		rec.Services = make(map[int]*serviceRecord)
	}
	rec.Tailnet = true
	if peer.NodeID != "" {
		rec.TailnetID = peer.NodeID
	}
	if peer.IP != "" {
		if !rec.Local {
			rec.IP = peer.IP
		}
		rec.TailnetIP = peer.IP
	}
	if name := peerDisplayName(peer); name != "" && rec.Hostname == "" {
		rec.Hostname = name
	}
	rec.TailnetName = strings.TrimSuffix(strings.TrimSpace(peer.DNSName), ".")
	if peer.OS != "" {
		rec.TailnetOS = peer.OS
	}
	rec.TailnetTags = cleanTags(peer.Tags)
	if kind := peer.Kind; kind != "" && (rec.Kind == "" || rec.Kind == "device") {
		rec.Kind = kind
	}
	if peer.Self {
		// A stale tailnet-only self record may be merged later; keep the flag
		// consistent with the strongest observation.
		rec.Self = true
	}
	if rec.FirstSeen.IsZero() {
		rec.FirstSeen = now
	}
	if peer.Online {
		rec.TailnetLastSeen = now
		// Tailscale knows the peer is up even when every TCP probe failed.
		rec.LastSeen = now
	} else if !peer.LastSeen.IsZero() && peer.LastSeen.After(rec.LastSeen) {
		// Preserve the last time Tailscale saw an offline peer so history shows
		// a meaningful timestamp instead of "never".
		rec.LastSeen = peer.LastSeen
	}

	for _, svc := range peer.Services {
		sr := rec.Services[svc.Port]
		if sr == nil {
			sr = &serviceRecord{Port: svc.Port, FirstSeen: now}
			rec.Services[svc.Port] = sr
		}
		if !svc.FirstSeen.IsZero() && (sr.FirstSeen.IsZero() || svc.FirstSeen.Before(sr.FirstSeen)) {
			sr.FirstSeen = svc.FirstSeen
		}
		if svc.Protocol != "" && sr.Protocol == "" {
			sr.Protocol = svc.Protocol
		}
		if svc.Name != "" && sr.Name == "" {
			sr.Name = svc.Name
		}
		if svc.Product != "" && sr.Product == "" {
			sr.Product = svc.Product
		}
		if svc.Category != "" && sr.Category == "" {
			sr.Category = svc.Category
		}
		if svc.Source != "" && sr.Source == "" {
			sr.Source = svc.Source
		}
		sr.Tailnet = true
		sr.LastSeen = now
	}
}

// mergeRecordLocked keeps target (the LAN record) and folds source into it.
// Services merge by port so history and both reach flags survive.
func (s *Store) mergeRecordLocked(target, source *deviceRecord) {
	if target == nil || source == nil || target == source {
		return
	}
	if target.Services == nil {
		target.Services = make(map[int]*serviceRecord)
	}
	for port, src := range source.Services {
		dst := target.Services[port]
		if dst == nil {
			target.Services[port] = src
			continue
		}
		dst.Local = dst.Local || src.Local
		dst.Tailnet = dst.Tailnet || src.Tailnet
		if !src.FirstSeen.IsZero() && (dst.FirstSeen.IsZero() || src.FirstSeen.Before(dst.FirstSeen)) {
			dst.FirstSeen = src.FirstSeen
		}
		if src.LastSeen.After(dst.LastSeen) {
			dst.LastSeen = src.LastSeen
		}
		if dst.Protocol == "" {
			dst.Protocol = src.Protocol
		}
		if dst.Name == "" {
			dst.Name = src.Name
		}
		if dst.Product == "" {
			dst.Product = src.Product
		}
		if dst.Category == "" {
			dst.Category = src.Category
		}
		if dst.Source == "" {
			dst.Source = src.Source
		}
	}
	if target.FirstSeen.IsZero() || (!source.FirstSeen.IsZero() && source.FirstSeen.Before(target.FirstSeen)) {
		target.FirstSeen = source.FirstSeen
	}
	if source.LastSeen.After(target.LastSeen) {
		target.LastSeen = source.LastSeen
	}
	if source.LocalLastSeen.After(target.LocalLastSeen) {
		target.LocalLastSeen = source.LocalLastSeen
	}
	if source.TailnetLastSeen.After(target.TailnetLastSeen) {
		target.TailnetLastSeen = source.TailnetLastSeen
	}
	target.Local = target.Local || source.Local
	target.Tailnet = target.Tailnet || source.Tailnet
	target.Self = target.Self || source.Self
	target.Gateway = target.Gateway || source.Gateway
	if target.Hostname == "" {
		target.Hostname = source.Hostname
	}
	if target.MAC == "" {
		target.MAC = source.MAC
	}
	if target.Vendor == "" {
		target.Vendor = source.Vendor
	}
	if target.Kind == "" || target.Kind == "device" {
		if source.Kind != "" {
			target.Kind = source.Kind
		}
	}
	if target.TailnetID == "" {
		target.TailnetID = source.TailnetID
	}
	if target.TailnetIP == "" {
		target.TailnetIP = source.TailnetIP
	}
	if target.TailnetName == "" {
		target.TailnetName = source.TailnetName
	}
	if target.TailnetOS == "" {
		target.TailnetOS = source.TailnetOS
	}
	if len(target.TailnetTags) == 0 {
		target.TailnetTags = source.TailnetTags
	}
	delete(s.devices, source.ID)
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
			// A new device may reuse an address another MAC held before. Only
			// reuse the record when the address is not claimed by a different
			// known MAC.
			if rec.IP == ip && (mac == "" || rec.MAC == "" || rec.MAC == mac) {
				return key, rec
			}
		}
	}
	return "", nil
}

func (s *Store) findByTailnetIDLocked(id string) *deviceRecord {
	if id == "" {
		return nil
	}
	for _, rec := range s.devices {
		if rec.TailnetID == id {
			return rec
		}
	}
	return nil
}

// lanNameIndexLocked maps a normalized short hostname to LAN records. Only
// records that have been observed on the LAN participate in name matching.
func (s *Store) lanNameIndexLocked() map[string][]*deviceRecord {
	index := make(map[string][]*deviceRecord)
	for _, rec := range s.devices {
		if !rec.Local {
			continue
		}
		if name := shortName(rec.Hostname); name != "" {
			index[name] = append(index[name], rec)
		}
	}
	return index
}

// matchLANLocked returns the LAN record a peer belongs to, or nil when the
// match would be ambiguous.
func (s *Store) matchLANLocked(peer model.TailnetPeer, peerNameCounts map[string]int, index map[string][]*deviceRecord) *deviceRecord {
	if peer.Self {
		candidates := make([]*deviceRecord, 0, 1)
		for _, rec := range s.devices {
			if rec.Self && rec.Local {
				candidates = append(candidates, rec)
			}
		}
		if len(candidates) == 1 {
			return candidates[0]
		}
	}
	seen := make(map[*deviceRecord]struct{})
	var match *deviceRecord
	for _, name := range peerNames(peer) {
		if peerNameCounts[name] > 1 {
			// Two peers claim this name; refuse to guess.
			return nil
		}
		candidates := index[name]
		if len(candidates) != 1 {
			if len(candidates) > 1 {
				return nil
			}
			continue
		}
		if _, ok := seen[candidates[0]]; ok {
			continue
		}
		seen[candidates[0]] = struct{}{}
		match = candidates[0]
	}
	if len(seen) > 1 {
		return nil
	}
	return match
}

// peerNames returns the normalized short names a peer could be known by.
func peerNames(peer model.TailnetPeer) []string {
	var names []string
	for _, raw := range []string{peer.DNSName, peer.HostName} {
		name := shortName(raw)
		if name == "" {
			continue
		}
		duplicate := false
		for _, existing := range names {
			if existing == name {
				duplicate = true
				break
			}
		}
		if !duplicate {
			names = append(names, name)
		}
	}
	return names
}

// peerDisplayName prefers the MagicDNS label, which is the name Tailscale
// shows, and falls back to HostName unless it is the generic "localhost".
func peerDisplayName(peer model.TailnetPeer) string {
	if name := shortName(peer.DNSName); name != "" {
		return name
	}
	if name := shortName(peer.HostName); name != "" && name != "localhost" {
		return name
	}
	return ""
}

// shortName normalizes a hostname for matching: lower case, no trailing dot,
// no DNS suffix and no leading underscore from PTR records.
func shortName(raw string) string {
	name := strings.ToLower(strings.TrimSpace(raw))
	name = strings.TrimSuffix(name, ".")
	if dot := strings.IndexByte(name, '.'); dot >= 0 {
		name = name[:dot]
	}
	name = strings.TrimPrefix(name, "_")
	if name == "" || name == "localhost" {
		return ""
	}
	return name
}

func tailnetKey(peer model.TailnetPeer) string {
	id := strings.TrimSpace(peer.NodeID)
	if id == "" {
		id = strings.TrimSpace(peer.IP)
	}
	return "ts:" + id
}

func cleanTags(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, tag := range in {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			out = append(out, tag)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
	fresh := func(t time.Time) bool {
		return !t.IsZero() && now.Sub(t) <= onlineFor
	}

	devices := make([]model.Device, 0, len(s.devices))
	stats := model.Stats{}
	for _, rec := range s.devices {
		d := model.Device{
			ID:            rec.ID,
			IP:            rec.IP,
			MAC:           rec.MAC,
			Hostname:      rec.Hostname,
			Vendor:        rec.Vendor,
			Kind:          rec.Kind,
			Gateway:       rec.Gateway,
			Self:          rec.Self,
			FirstSeen:     rec.FirstSeen,
			LastSeen:      rec.LastSeen,
			Local:         rec.Local,
			Tailnet:       rec.Tailnet,
			TailnetID:     rec.TailnetID,
			TailnetIP:     rec.TailnetIP,
			TailnetName:   rec.TailnetName,
			TailnetOS:     rec.TailnetOS,
			TailnetTags:   append([]string(nil), rec.TailnetTags...),
			TailnetOnline: rec.Tailnet && fresh(rec.TailnetLastSeen),
		}
		if d.Kind == "" {
			d.Kind = "device"
		}
		d.Online = fresh(rec.LastSeen)

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
				Local:     sr.Local,
				Tailnet:   sr.Tailnet,
			}
			if svc.Protocol == "" {
				svc.Protocol = "tcp"
			}
			if svc.Category == "" {
				svc.Category = "other"
			}
			svc.Online = d.Online && fresh(sr.LastSeen)
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

	var tailnet *model.Tailnet
	if s.tailnet.Name != "" || s.tailnet.MagicDNSSuffix != "" || s.tailnet.PeersTotal > 0 {
		t := s.tailnet
		tailnet = &t
	}

	return model.Snapshot{
		GeneratedAt:     now,
		LastScan:        s.lastScan,
		Scanning:        opts.Scanning,
		ScanIntervalSec: int(opts.Interval / time.Second),
		ScanDurationMS:  opts.ScanDuration.Milliseconds(),
		Networks:        append([]string(nil), s.networks...),
		LastError:       opts.LastError,
		Tailnet:         tailnet,
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
	if s.tailnet.Name != "" || s.tailnet.MagicDNSSuffix != "" || s.tailnet.PeersTotal > 0 {
		t := s.tailnet
		p.Tailnet = &t
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
