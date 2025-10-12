package life

import (
	"mad-ca/pkg/core"
)

// Life implements Conway's Game of Life with toroidal wrapping.
type Life struct {
	w, h int
	cur  []uint8
	nxt  []uint8
}

// New returns a Life simulation with the provided dimensions.
func New(w, h int) *Life {
	cells := make([]uint8, w*h)
	return &Life{w: w, h: h, cur: cells, nxt: make([]uint8, len(cells))}
}

// Name returns the simulation identifier.
func (l *Life) Name() string { return "life" }

// Size returns the grid dimensions.
func (l *Life) Size() core.Size { return core.Size{W: l.w, H: l.h} }

// Cells exposes the current grid values.
func (l *Life) Cells() []uint8 { return l.cur }

// Reset randomizes the board using the provided seed.
func (l *Life) Reset(seed int64) {
	rng := core.NewRNG(seed).Source()
	core.FillBinary(rng, l.cur)
}

// Step advances the simulation by one generation.
func (l *Life) Step() {
	w, h := l.w, l.h
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			neighbors := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx := (x + dx + w) % w
					ny := (y + dy + h) % h
					neighbors += int(l.cur[ny*w+nx])
				}
			}
			idx := y*w + x
			alive := l.cur[idx] == 1
			l.nxt[idx] = 0
			if (alive && (neighbors == 2 || neighbors == 3)) || (!alive && neighbors == 3) {
				l.nxt[idx] = 1
			}
		}
	}
	l.cur, l.nxt = l.nxt, l.cur
}

func init() {
	core.Register("life", func(cfg map[string]string) core.Sim {
		c := FromMap(cfg)
		return New(c.Width, c.Height)
	})
}
