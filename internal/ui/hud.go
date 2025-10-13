//go:build ebiten

package ui

import (
	"image/color"

	"mad-ca/internal/core"

	"github.com/hajimehoshi/ebiten/v2"
)

type parameterProvider interface {
	Parameters() core.ParameterSnapshot
}

// HUD renders the parameter panel to the right of the simulation view.
type HUD struct {
	sim        core.Sim
	width      int
	panel      *ebiten.Image
	lastHeight int
	snapshot   core.ParameterSnapshot
}

// NewHUD constructs a HUD for the provided simulation and panel width.
func NewHUD(sim core.Sim, width int) *HUD {
	if width < 0 {
		width = 0
	}
	return &HUD{sim: sim, width: width}
}

// Update refreshes the cached parameter snapshot from the simulation.
func (h *HUD) Update() {
	if h == nil {
		return
	}
	provider, ok := h.sim.(parameterProvider)
	if !ok {
		h.snapshot = core.ParameterSnapshot{}
		return
	}
	h.snapshot = provider.Parameters()
}

// Draw paints the HUD panel anchored to the right edge of the simulation view.
func (h *HUD) Draw(screen *ebiten.Image, offsetX int, scale int) {
	if h == nil || h.width <= 0 {
		return
	}
	if scale <= 0 {
		scale = 1
	}
	size := h.sim.Size()
	height := size.H * scale
	if height <= 0 {
		return
	}
	if h.panel == nil || h.panel.Bounds().Dx() != h.width || h.lastHeight != height {
		h.panel = ebiten.NewImage(h.width, height)
		h.panel.Fill(color.Black)
		h.lastHeight = height
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(offsetX), 0)
	screen.DrawImage(h.panel, op)
}
