//go:build ebiten

package app

import (
	"image/color"
	"math"
	"time"

	"mad-ca/internal/core"
	"mad-ca/internal/render"
	"mad-ca/internal/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Igniter allows simulations to expose manual fire ignition hooks for debugging.
type Igniter interface {
	IgniteAt(x, y int)
}

// PaletteProvider allows simulations to expose a color palette for rendering
// multi-valued cell buffers. When unavailable the renderer falls back to the
// binary on/off colors.
type PaletteProvider interface {
	Palette() []color.RGBA
}

// Game adapts a core simulation to the ebiten.Game interface.
type Game struct {
	sim     core.Sim
	painter *render.GridPainter
	overlay *ui.Overlay
	hud     *ui.HUD

	onColor  color.Color
	offColor color.Color

	scale    int
	paused   bool
	tickOnce bool
	seed     int64

	igniter  Igniter
	hudWidth int
}

// New constructs a Game for the provided simulation.
func New(sim core.Sim, scale int, seed int64) *Game {
	if scale <= 0 {
		scale = 1
	}
	size := sim.Size()
	gp := render.NewGridPainter(size.W, size.H)
	var igniter Igniter
	if i, ok := sim.(Igniter); ok {
		igniter = i
	}
	baseWidth := size.W * scale
	hudWidth := int(math.Round(float64(baseWidth) / 4.0))
	if baseWidth > 0 && hudWidth == 0 {
		hudWidth = 1
	}
	return &Game{
		sim:      sim,
		painter:  gp,
		overlay:  ui.NewOverlay(sim, scale),
		hud:      ui.NewHUD(sim, hudWidth),
		onColor:  color.White,
		offColor: color.Black,
		scale:    scale,
		seed:     seed,
		igniter:  igniter,
		hudWidth: hudWidth,
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
	if g.hud != nil {
		g.hud.Update()
	}

	if g.igniter != nil && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		scale := g.scale
		if scale <= 0 {
			scale = 1
		}
		g.igniter.IgniteAt(mx/scale, my/scale)
	}

	if (!g.paused) || g.tickOnce {
		g.sim.Step()
		g.tickOnce = false
	}
	return nil
}

// Draw renders the current simulation state.
func (g *Game) Draw(screen *ebiten.Image) {
	if provider, ok := g.sim.(PaletteProvider); ok {
		g.painter.BlitPalette(screen, g.sim.Cells(), provider.Palette(), g.scale)
	} else {
		g.painter.Blit(screen, g.sim.Cells(), g.onColor, g.offColor, g.scale)
	}
	if g.overlay != nil {
		g.overlay.Draw(screen)
	}
	if g.hud != nil {
		baseWidth := g.sim.Size().W * g.scale
		g.hud.Draw(screen, baseWidth, g.scale)
	}
}

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := g.sim.Size()
	return s.W*g.scale + g.hudWidth, s.H * g.scale
}
