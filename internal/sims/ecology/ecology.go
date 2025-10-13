package ecology

import "mad-ca/internal/core"

// World is a placeholder simulation that keeps an empty grid stable.
type World struct {
	w, h  int
	cells []uint8
}

// New returns an Ecology simulation with the provided dimensions.
func New(w, h int) *World {
	return &World{w: w, h: h, cells: make([]uint8, w*h)}
}

// Name returns the simulation identifier.
func (w *World) Name() string { return "ecology" }

// Size reports the grid dimensions.
func (w *World) Size() core.Size { return core.Size{W: w.w, H: w.h} }

// Cells exposes the current grid values.
func (w *World) Cells() []uint8 { return w.cells }

// Reset clears the grid, producing an empty world regardless of the seed.
func (w *World) Reset(seed int64) {
	clear(w.cells)
}

// Step advances the simulation. Ecology has no rules so the grid never changes.
func (w *World) Step() {}

func init() {
	core.Register("ecology", func(cfg map[string]string) core.Sim {
		c := FromMap(cfg)
		return New(c.Width, c.Height)
	})
}
