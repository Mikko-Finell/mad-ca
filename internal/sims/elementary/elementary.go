package elementary

import (
	"strconv"

	"mad-ca/internal/core"
)

// Config holds parameters for the elementary cellular automaton.
type Config struct {
	Width  int
	Height int
	Rule   uint8
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{Width: 256, Height: 256, Rule: 110}
}

// FromMap populates a Config from a string map.
func FromMap(cfg map[string]string) Config {
	c := DefaultConfig()
	if cfg == nil {
		return c
	}
	if v, ok := cfg["w"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			c.Width = parsed
		}
	}
	if v, ok := cfg["h"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			c.Height = parsed
		}
	}
	if v, ok := cfg["rule"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 && parsed <= 255 {
			c.Rule = uint8(parsed)
		}
	}
	return c
}

// Elementary implements a one-dimensional Wolfram code projected vertically.
type Elementary struct {
	w, h int
	rule uint8
	cur  []uint8
	tmp  []uint8
}

// New creates an automaton with the given dimensions and rule.
func New(w, h int, rule uint8) *Elementary {
	total := w * h
	return &Elementary{w: w, h: h, rule: rule, cur: make([]uint8, total), tmp: make([]uint8, w)}
}

// Name returns the simulation identifier.
func (e *Elementary) Name() string { return "elementary" }

// Size returns the simulation grid dimensions.
func (e *Elementary) Size() core.Size { return core.Size{W: e.w, H: e.h} }

// Cells exposes the render buffer.
func (e *Elementary) Cells() []uint8 { return e.cur }

// Reset clears the grid and seeds the top row with a single active cell.
func (e *Elementary) Reset(seed int64) {
	for i := range e.cur {
		e.cur[i] = 0
	}
	center := e.w / 2
	if center >= 0 && center < e.w {
		e.cur[center] = 1
	}
}

// Step computes the next generation and scrolls history downwards.
func (e *Elementary) Step() {
	copy(e.tmp, e.cur[:e.w])
	copy(e.cur[e.w:], e.cur[:e.w*(e.h-1)])
	for x := 0; x < e.w; x++ {
		left := e.tmp[(x-1+e.w)%e.w]
		center := e.tmp[x]
		right := e.tmp[(x+1)%e.w]
		idx := (left << 2) | (center << 1) | right
		bit := (e.rule >> idx) & 1
		e.cur[x] = bit
	}
}

func init() {
	core.Register("elementary", func(cfg map[string]string) core.Sim {
		c := FromMap(cfg)
		return New(c.Width, c.Height, c.Rule)
	})
}
