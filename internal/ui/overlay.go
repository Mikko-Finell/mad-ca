//go:build ebiten

package ui

import (
	"image/color"
	"math"

	"mad-ca/internal/core"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type maskProvider interface {
	RainMask() []float32
	VolcanoMask() []float32
}

// Overlay draws optional debugging visuals on top of the base simulation.
type Overlay struct {
	sim         core.Sim
	scale       int
	showRain    bool
	showVolcano bool
	maskImg     *ebiten.Image
	maskBuf     []byte
}

// NewOverlay constructs a new overlay instance.
func NewOverlay(sim core.Sim, scale int) *Overlay {
	return &Overlay{sim: sim, scale: scale}
}

// Update allows the overlay to update internal state.
func (o *Overlay) Update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		o.showRain = !o.showRain
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		o.showVolcano = !o.showVolcano
	}
}

// Draw renders the overlay onto the provided screen.
func (o *Overlay) Draw(screen *ebiten.Image) {
	provider, ok := o.sim.(maskProvider)
	if !ok {
		return
	}
	size := o.sim.Size()
	if size.W <= 0 || size.H <= 0 {
		return
	}
	total := size.W * size.H
	if total == 0 {
		return
	}
	if o.maskImg == nil || o.maskImg.Bounds().Dx() != size.W || o.maskImg.Bounds().Dy() != size.H {
		o.maskImg = ebiten.NewImage(size.W, size.H)
		o.maskBuf = make([]byte, 4*total)
	} else if len(o.maskBuf) != 4*total {
		o.maskBuf = make([]byte, 4*total)
	}

	if o.showRain {
		o.drawMask(screen, provider.RainMask(), color.RGBA{R: 64, G: 164, B: 223, A: 0})
	}
	if o.showVolcano {
		o.drawMask(screen, provider.VolcanoMask(), color.RGBA{R: 255, G: 120, B: 40, A: 0})
	}
}

func (o *Overlay) drawMask(screen *ebiten.Image, mask []float32, tint color.RGBA) {
	size := o.sim.Size()
	total := size.W * size.H
	if len(mask) != total {
		return
	}
	const (
		maxAlpha      = 140.0
		glowBase      = 0.35
		glowRange     = 0.65
		intensityBias = 0.75
	)

	for i := 0; i < total; i++ {
		base := i * 4
		intensity := float64(mask[i])
		if intensity < 0 {
			intensity = 0
		}
		if intensity > 1 {
			intensity = 1
		}
		if intensity == 0 {
			o.maskBuf[base+0] = 0
			o.maskBuf[base+1] = 0
			o.maskBuf[base+2] = 0
			o.maskBuf[base+3] = 0
			continue
		}

		alpha := uint8(math.Round(maxAlpha * math.Pow(intensity, intensityBias)))
		glow := glowBase + glowRange*math.Sqrt(intensity)

		o.maskBuf[base+0] = scaleColorComponent(tint.R, glow)
		o.maskBuf[base+1] = scaleColorComponent(tint.G, glow)
		o.maskBuf[base+2] = scaleColorComponent(tint.B, glow)
		o.maskBuf[base+3] = alpha
	}
	o.maskImg.ReplacePixels(o.maskBuf)
	op := &ebiten.DrawImageOptions{}
	scale := o.scale
	if scale <= 0 {
		scale = 1
	}
	op.GeoM.Scale(float64(scale), float64(scale))
	screen.DrawImage(o.maskImg, op)
}

func scaleColorComponent(value uint8, factor float64) uint8 {
	scaled := math.Round(float64(value) * factor)
	if scaled < 0 {
		return 0
	}
	if scaled > 255 {
		return 255
	}
	return uint8(scaled)
}
