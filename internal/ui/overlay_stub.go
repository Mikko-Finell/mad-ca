//go:build !ebiten

package ui

import "mad-ca/internal/core"

// Overlay is a no-op placeholder used when the ebiten build tag is absent.
type Overlay struct{}

// NewOverlay constructs a stub overlay.
func NewOverlay(core.Sim, int) *Overlay { return &Overlay{} }

// Update is a no-op in headless builds.
func (o *Overlay) Update() {}

// Draw is a no-op placeholder.
func (o *Overlay) Draw(any) {}
