package traffic

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jmkelly/hearthlight/internal/discovery"
	"github.com/jmkelly/hearthlight/internal/model"
)

var devices = []model.Device{
	{ID: "self", IP: "192.0.2.58", Local: true, Self: true, TailnetIP: "100.101.102.103", TailnetID: "nSelf"},
	{ID: "nas", IP: "192.0.2.20", Local: true, Tailnet: true, TailnetIP: "100.101.102.18", TailnetID: "nNas"},
	{ID: "tv", IP: "192.0.2.42", Local: true},
	{ID: "ts:cloud", IP: "100.101.102.9", Tailnet: true, TailnetIP: "100.101.102.9", TailnetID: "nCloud"},
}

func conn(local, remote string) discovery.TCPConn {
	return discovery.TCPConn{Local: netip.MustParseAddrPort(local), Remote: netip.MustParseAddrPort(remote)}
}

func newTestSampler(conns []discovery.TCPConn, counters *map[string]discovery.TailnetCounter) *Sampler {
	s := New(true, nil)
	s.readConns = func() ([]discovery.TCPConn, error) { return conns, nil }
	s.readCounters = func(context.Context) (map[string]discovery.TailnetCounter, error) {
		if *counters == nil {
			return nil, errors.New("tailscale not running")
		}
		return *counters, nil
	}
	return s
}

func find(flows []model.Flow, id, via string) *model.Flow {
	for i := range flows {
		if flows[i].DeviceID == id && flows[i].Via == via {
			return &flows[i]
		}
	}
	return nil
}

func TestSampleAttributesConnectionsByNetwork(t *testing.T) {
	var counters map[string]discovery.TailnetCounter
	s := newTestSampler([]discovery.TCPConn{
		conn("192.0.2.58:50000", "192.0.2.20:445"),
		conn("192.0.2.58:50001", "192.0.2.20:445"),
		conn("100.101.102.103:50002", "100.101.102.18:22"),
		conn("192.0.2.58:50003", "192.0.2.42:8009"),
		conn("192.0.2.58:50004", "203.0.113.9:443"), // internet, not on the map
		conn("127.0.0.1:8090", "127.0.0.1:60000"),   // loopback
		conn("192.0.2.58:8090", "192.0.2.58:60001"), // this host
	}, &counters)

	flows := s.Sample(context.Background(), devices, time.Now(), false)
	if len(flows) != 3 {
		t.Fatalf("got %d flows, want 3: %+v", len(flows), flows)
	}
	if f := find(flows, "nas", "lan"); f == nil || f.Conns != 2 {
		t.Errorf("nas lan = %+v, want 2 conns", f)
	}
	if f := find(flows, "nas", "tailnet"); f == nil || f.Conns != 1 || f.Measured {
		t.Errorf("nas tailnet = %+v, want 1 conn, unmeasured", f)
	}
	if f := find(flows, "tv", "lan"); f == nil || f.Conns != 1 {
		t.Errorf("tv lan = %+v", f)
	}
}

func TestSampleComputesTailnetRates(t *testing.T) {
	counters := map[string]discovery.TailnetCounter{
		"nCloud": {RxBytes: 1000, TxBytes: 500},
		"nNas":   {RxBytes: 50, TxBytes: 50},
	}
	s := newTestSampler(nil, &counters)
	start := time.Now()
	if flows := s.Sample(context.Background(), devices, start, false); len(flows) != 0 {
		t.Fatalf("first sample has no baseline, got %+v", flows)
	}

	counters = map[string]discovery.TailnetCounter{
		"nCloud": {RxBytes: 5000, TxBytes: 1500},
		"nNas":   {RxBytes: 50, TxBytes: 50},
	}
	flows := s.Sample(context.Background(), devices, start.Add(2*time.Second), false)
	f := find(flows, "ts:cloud", "tailnet")
	if f == nil || f.RxBps != 2000 || f.TxBps != 500 || !f.Measured {
		t.Fatalf("cloud = %+v, want rx 2000 tx 500", f)
	}
	if f := find(flows, "nas", "tailnet"); f != nil {
		t.Errorf("idle peer should have no flow, got %+v", f)
	}

	// A WireGuard session restart resets the counters; that must not produce
	// a negative or huge rate.
	counters = map[string]discovery.TailnetCounter{"nCloud": {RxBytes: 10, TxBytes: 10}}
	flows = s.Sample(context.Background(), devices, start.Add(4*time.Second), false)
	if f := find(flows, "ts:cloud", "tailnet"); f != nil {
		t.Errorf("reset counters should give no rate, got %+v", f)
	}

	// Losing tailscaled drops the baseline instead of reusing a stale one.
	counters = nil
	s.Sample(context.Background(), devices, start.Add(6*time.Second), false)
	counters = map[string]discovery.TailnetCounter{"nCloud": {RxBytes: 9999, TxBytes: 9999}}
	if flows := s.Sample(context.Background(), devices, start.Add(8*time.Second), false); len(flows) != 0 {
		t.Errorf("first sample after an outage has no baseline, got %+v", flows)
	}
}

func TestSampleHoldsConnectionsDuringScan(t *testing.T) {
	var counters map[string]discovery.TailnetCounter
	conns := []discovery.TCPConn{conn("192.0.2.58:50000", "192.0.2.42:8009")}
	s := newTestSampler(nil, &counters)
	s.readConns = func() ([]discovery.TCPConn, error) { return conns, nil }
	s.Sample(context.Background(), devices, time.Now(), false)

	// During a scan the probe connections must not appear as traffic.
	conns = []discovery.TCPConn{
		conn("192.0.2.58:50010", "192.0.2.20:22"),
		conn("192.0.2.58:50011", "192.0.2.20:80"),
	}
	flows := s.Sample(context.Background(), devices, time.Now(), true)
	if len(flows) != 1 || flows[0].DeviceID != "tv" {
		t.Fatalf("held sample should reuse the previous table, got %+v", flows)
	}
}
