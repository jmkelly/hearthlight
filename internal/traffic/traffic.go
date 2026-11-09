// Package traffic samples live traffic between this host and the devices on
// the map. It sees only this host's own traffic: on a switched network no
// unprivileged host can observe traffic between two other devices.
package traffic

import (
	"context"
	"log/slog"
	"net/netip"
	"sort"
	"time"

	"github.com/jmkelly/hearthlight/internal/discovery"
	"github.com/jmkelly/hearthlight/internal/model"
)

var cgnatPrefix = netip.MustParsePrefix("100.64.0.0/10")

// Sampler turns connection tables and Tailscale byte counters into flows.
// It is not safe for concurrent use; one loop owns it.
type Sampler struct {
	tailnet bool
	logger  *slog.Logger

	readConns    func() ([]discovery.TCPConn, error)
	readCounters func(context.Context) (map[string]discovery.TailnetCounter, error)

	lastConns   []discovery.TCPConn
	prev        map[string]discovery.TailnetCounter
	prevAt      time.Time
	connsFailed bool
	tsFailed    bool
}

// New returns a sampler. When tailnet is false, Tailscale is never queried.
func New(tailnet bool, logger *slog.Logger) *Sampler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Sampler{
		tailnet:      tailnet,
		logger:       logger,
		readConns:    discovery.ReadTCPConnections,
		readCounters: discovery.ReadTailnetCounters,
	}
}

// Sample reads the current traffic and attributes it to devices. While
// holdConns is true the previous connection table is reused: a scan opens
// short-lived probe connections to every host, and those are not traffic.
func (s *Sampler) Sample(ctx context.Context, devices []model.Device, now time.Time, holdConns bool) []model.Flow {
	idx := buildIndex(devices)
	flows := make(map[flowKey]*model.Flow)
	flow := func(id, via string) *model.Flow {
		key := flowKey{id, via}
		f := flows[key]
		if f == nil {
			f = &model.Flow{DeviceID: id, Via: via}
			flows[key] = f
		}
		return f
	}

	if !holdConns {
		conns, err := s.readConns()
		switch {
		case err != nil && !s.connsFailed:
			s.logger.Warn("traffic: cannot read TCP connections", "error", err)
			s.connsFailed = true
		case err == nil && s.connsFailed:
			s.logger.Info("traffic: TCP connections readable again")
			s.connsFailed = false
		}
		s.lastConns = conns
	}
	for _, c := range s.lastConns {
		remote := c.Remote.Addr().Unmap()
		if cgnatPrefix.Contains(remote) {
			if id, ok := idx.tailnetIP[remote]; ok {
				flow(id, "tailnet").Conns++
			}
			continue
		}
		if id, ok := idx.lanIP[remote]; ok {
			flow(id, "lan").Conns++
		}
	}

	if s.tailnet {
		s.sampleCounters(ctx, idx, now, flow)
	}

	out := make([]model.Flow, 0, len(flows))
	for _, f := range flows {
		if f.Conns > 0 || f.RxBps > 0 || f.TxBps > 0 {
			out = append(out, *f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DeviceID != out[j].DeviceID {
			return out[i].DeviceID < out[j].DeviceID
		}
		return out[i].Via < out[j].Via
	})
	return out
}

func (s *Sampler) sampleCounters(ctx context.Context, idx index, now time.Time, flow func(id, via string) *model.Flow) {
	counters, err := s.readCounters(ctx)
	if err != nil {
		if !s.tsFailed {
			s.logger.Warn("traffic: tailnet counters unavailable", "error", err)
			s.tsFailed = true
		}
		s.prev = nil
		return
	}
	if s.tsFailed {
		s.logger.Info("traffic: tailnet counters available again")
		s.tsFailed = false
	}
	prev, prevAt := s.prev, s.prevAt
	s.prev, s.prevAt = counters, now
	elapsed := now.Sub(prevAt).Seconds()
	if prev == nil || elapsed <= 0 {
		return
	}
	for nodeID, cur := range counters {
		id, ok := idx.tailnetID[nodeID]
		if !ok {
			continue
		}
		f := flow(id, "tailnet")
		f.Measured = true
		old, ok := prev[nodeID]
		if !ok || cur.RxBytes < old.RxBytes || cur.TxBytes < old.TxBytes {
			// New peer or a restarted WireGuard session: no rate this round.
			continue
		}
		f.RxBps = float64(cur.RxBytes-old.RxBytes) / elapsed
		f.TxBps = float64(cur.TxBytes-old.TxBytes) / elapsed
	}
}

type flowKey struct{ id, via string }

type index struct {
	lanIP     map[netip.Addr]string
	tailnetIP map[netip.Addr]string
	tailnetID map[string]string
}

// buildIndex maps addresses and Tailscale node IDs to device IDs. This host's
// own device is left out so it never gets an edge to itself.
func buildIndex(devices []model.Device) index {
	idx := index{
		lanIP:     make(map[netip.Addr]string),
		tailnetIP: make(map[netip.Addr]string),
		tailnetID: make(map[string]string),
	}
	for _, d := range devices {
		if d.Self {
			continue
		}
		if addr, err := netip.ParseAddr(d.IP); err == nil && d.Local && !cgnatPrefix.Contains(addr) {
			idx.lanIP[addr] = d.ID
		}
		if addr, err := netip.ParseAddr(d.TailnetIP); err == nil {
			idx.tailnetIP[addr] = d.ID
		}
		if d.TailnetID != "" {
			idx.tailnetID[d.TailnetID] = d.ID
		}
	}
	return idx
}
