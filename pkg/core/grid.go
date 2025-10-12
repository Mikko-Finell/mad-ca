package core

// ByteGrid stores a 2D grid of byte-sized cell values in row-major order.
type ByteGrid struct {
	W, H int
	data []uint8
}

// NewByteGrid allocates a grid with the given dimensions.
func NewByteGrid(w, h int) *ByteGrid {
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	return &ByteGrid{W: w, H: h, data: make([]uint8, w*h)}
}

// Cells exposes the backing slice so callers can read/write values directly.
func (g *ByteGrid) Cells() []uint8 { return g.data }

// Index returns the linear slice index for coordinates (x, y).
func (g *ByteGrid) Index(x, y int) int { return y*g.W + x }

// Wrap applies toroidal wrapping to the provided coordinates.
func (g *ByteGrid) Wrap(x, y int) (int, int) {
	x = (x%g.W + g.W) % g.W
	y = (y%g.H + g.H) % g.H
	return x, y
}

// Clear fills the grid with zeros.
func (g *ByteGrid) Clear() {
	for i := range g.data {
		g.data[i] = 0
	}
}
