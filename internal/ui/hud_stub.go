//go:build !ebiten

package ui

import "mad-ca/internal/core"

// HUD is a no-op placeholder for headless builds.
type HUD struct{}

// NewHUD returns nil in the headless build.
func NewHUD(core.Sim, int) *HUD { return nil }

// Update is a no-op in the headless build.
func (h *HUD) Update() {}

// Draw is a no-op in the headless build.
func (h *HUD) Draw(any, int, int) {}
