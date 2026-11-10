package demo

import (
	"sync"

	"github.com/jmkelly/hearthlight/internal/model"
)

type point struct {
	x float64
	y float64
}

// Layout keeps manual map positions for the demo without writing a state
// file. It lets the drag-and-pin flow work in `make demo` as it does on a
// real inventory.
type Layout struct {
	mu        sync.Mutex
	positions map[string]point
}

// NewLayout returns an empty, in-memory demo layout.
func NewLayout() *Layout {
	return &Layout{positions: make(map[string]point)}
}

// SetDevicePosition records a pinned position for a demo device.
func (l *Layout) SetDevicePosition(id string, x, y float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.positions[id] = point{x: x, y: y}
	return nil
}

// ClearDevicePositions drops every manual position.
func (l *Layout) ClearDevicePositions() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	clear(l.positions)
	return nil
}

// Apply overlays saved positions on a freshly generated demo snapshot.
func (l *Layout) Apply(snapshot model.Snapshot) model.Snapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.positions) == 0 {
		return snapshot
	}
	snapshot.Devices = append([]model.Device(nil), snapshot.Devices...)
	for i := range snapshot.Devices {
		position, ok := l.positions[snapshot.Devices[i].ID]
		if !ok {
			continue
		}
		snapshot.Devices[i].X = position.x
		snapshot.Devices[i].Y = position.y
		snapshot.Devices[i].Pinned = true
	}
	return snapshot
}
