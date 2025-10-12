//go:build !ebiten

package app

import (
	"fmt"

	"mad-ca/internal/core"
)

// Game is a placeholder that satisfies the API expected by the GUI build.
type Game struct{}

// New panics to indicate that the ebiten build tag is required for GUI support.
func New(core.Sim, int, int64) *Game {
	panic("app.New requires building with the 'ebiten' tag")
}

// Reset is a no-op placeholder.
func (g *Game) Reset(int64) {}

// Update always reports that the GUI build tag is missing.
func (g *Game) Update() error {
	return fmt.Errorf("app.Game.Update requires building with the 'ebiten' tag")
}

// Draw is a no-op placeholder to satisfy the interface shape.
func (g *Game) Draw(any) {}

// Layout returns zeros in the headless build.
func (g *Game) Layout(int, int) (int, int) { return 0, 0 }
