//go:build ebiten

package app

import (
	"image/color"
	"time"

	"mad-ca/internal/core"
	"mad-ca/internal/render"
	"mad-ca/internal/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game adapts a core simulation to the ebiten.Game interface.
type Game struct {
	sim     core.Sim
	painter *render.GridPainter
	overlay *ui.Overlay

	onColor  color.Color
	offColor color.Color

	scale    int
	paused   bool
	tickOnce bool
	seed     int64
}

// New constructs a Game for the provided simulation.
func New(sim core.Sim, scale int, seed int64) *Game {
	gp := render.NewGridPainter(sim.Size().W, sim.Size().H)
	return &Game{
		sim:      sim,
		painter:  gp,
		overlay:  ui.NewOverlay(),
		onColor:  color.White,
		offColor: color.Black,
		scale:    scale,
		seed:     seed,
	}
}

// Reset reinitializes the simulation state with the provided seed.
func (g *Game) Reset(seed int64) {
	g.seed = seed
	g.sim.Reset(seed)
	g.tickOnce = false
}

// Update handles per-frame logic and advances the simulation.
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.paused = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.tickOnce = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Reset(g.seed)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.Reset(time.Now().UnixNano())
	}

	if g.overlay != nil {
		g.overlay.Update()
	}

	if (!g.paused) || g.tickOnce {
		g.sim.Step()
		g.tickOnce = false
	}
	return nil
}

// Draw renders the current simulation state.
func (g *Game) Draw(screen *ebiten.Image) {
	g.painter.Blit(screen, g.sim.Cells(), g.onColor, g.offColor, g.scale)
	if g.overlay != nil {
		g.overlay.Draw(screen)
	}
}

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := g.sim.Size()
	return s.W * g.scale, s.H * g.scale
}
