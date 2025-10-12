package briansbrain

import "mad-ca/internal/core"

const (
	stateDead  = 0
	stateOn    = 1
	stateDying = 2
)

// Brain implements Brian's Brain cellular automaton.
type Brain struct {
	w, h int
	cur  []uint8
	nxt  []uint8
}

// New creates a Brain simulation with the provided dimensions.
func New(w, h int) *Brain {
	cells := make([]uint8, w*h)
	return &Brain{w: w, h: h, cur: cells, nxt: make([]uint8, len(cells))}
}

// Name identifies the simulation.
func (b *Brain) Name() string { return "briansbrain" }

// Size returns the grid dimensions.
func (b *Brain) Size() core.Size { return core.Size{W: b.w, H: b.h} }

// Cells exposes the current state buffer.
func (b *Brain) Cells() []uint8 { return b.cur }

// Reset randomizes cells into dead or firing states.
func (b *Brain) Reset(seed int64) {
	rng := core.NewRNG(seed).Source()
	for i := range b.cur {
		if rng.IntN(8) == 0 {
			b.cur[i] = stateOn
			continue
		}
		b.cur[i] = stateDead
	}
}

// Step advances the automaton by one tick.
func (b *Brain) Step() {
	w, h := b.w, b.h
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			switch b.cur[idx] {
			case stateOn:
				b.nxt[idx] = stateDying
			case stateDying:
				b.nxt[idx] = stateDead
			default:
				neighbors := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx := (x + dx + w) % w
						ny := (y + dy + h) % h
						if b.cur[ny*w+nx] == stateOn {
							neighbors++
						}
					}
				}
				if neighbors == 2 {
					b.nxt[idx] = stateOn
				} else {
					b.nxt[idx] = stateDead
				}
			}
		}
	}
	b.cur, b.nxt = b.nxt, b.cur
}

func init() {
	core.Register("briansbrain", func(cfg map[string]string) core.Sim {
		return New(256, 256)
	})
}
