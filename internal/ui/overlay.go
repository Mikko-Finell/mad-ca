//go:build ebiten

package ui

import "github.com/hajimehoshi/ebiten/v2"

// Overlay is a placeholder for future UI elements (FPS counters, etc.).
type Overlay struct{}

// NewOverlay constructs a new overlay instance.
func NewOverlay() *Overlay { return &Overlay{} }

// Update allows the overlay to update internal state.
func (o *Overlay) Update() {}

// Draw renders the overlay onto the provided screen.
func (o *Overlay) Draw(screen *ebiten.Image) {}
