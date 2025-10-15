package ecology

import (
	"image/color"
	"math"
)

const (
	displayGroundMask      = 0x03
	displayVegetationShift = 2
	displayVegetationMask  = 0x0c
	displayBurningBit      = 0x10
)

var ecologyPalette = buildEcologyPalette()

// Palette exposes the color palette used for rendering the ecology world.
func (w *World) Palette() []color.RGBA {
	return ecologyPalette
}

func buildEcologyPalette() []color.RGBA {
	palette := make([]color.RGBA, 32)
	for i := range palette {
		ground := Ground(i & displayGroundMask)
		veg := Vegetation((i & displayVegetationMask) >> displayVegetationShift)
		burning := (i & displayBurningBit) != 0
		palette[i] = toRGBA(paletteColorFor(ground, veg, burning))
	}
	return palette
}

func toRGBA(c color.NRGBA) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

func paletteColorFor(ground Ground, veg Vegetation, burning bool) color.NRGBA {
	if burning {
		return color.NRGBA{R: 255, G: 130, B: 40, A: 255}
	}

	switch ground {
	case GroundLava:
		return color.NRGBA{R: 255, G: 90, B: 40, A: 255}
	case GroundMountain:
		base := color.NRGBA{R: 180, G: 180, B: 200, A: 255}
		if veg != VegetationNone {
			return blendColors(base, vegetationColor(veg), 0.55)
		}
		return base
	case GroundRock:
		base := color.NRGBA{R: 130, G: 130, B: 130, A: 255}
		if veg != VegetationNone {
			return blendColors(base, vegetationColor(veg), 0.65)
		}
		return base
	case GroundDirt:
		fallthrough
	default:
		base := color.NRGBA{R: 70, G: 52, B: 32, A: 255}
		if veg != VegetationNone {
			return blendColors(base, vegetationColor(veg), 0.75)
		}
		return base
	}
}

func vegetationColor(veg Vegetation) color.NRGBA {
	switch veg {
	case VegetationGrass:
		return color.NRGBA{R: 70, G: 160, B: 80, A: 255}
	case VegetationShrub:
		return color.NRGBA{R: 60, G: 125, B: 60, A: 255}
	case VegetationTree:
		return color.NRGBA{R: 40, G: 100, B: 55, A: 255}
	default:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	}
}

func blendColors(base, overlay color.NRGBA, overlayWeight float64) color.NRGBA {
	if overlayWeight <= 0 {
		return base
	}
	if overlayWeight >= 1 {
		return overlay
	}
	br, bg, bb, ba := float64(base.R), float64(base.G), float64(base.B), float64(base.A)
	or, og, ob, oa := float64(overlay.R), float64(overlay.G), float64(overlay.B), float64(overlay.A)
	w := overlayWeight
	inv := 1 - w
	return color.NRGBA{
		R: uint8(br*inv + or*w + 0.5),
		G: uint8(bg*inv + og*w + 0.5),
		B: uint8(bb*inv + ob*w + 0.5),
		A: uint8(ba*inv + oa*w + 0.5),
	}
}

func encodeDisplayValue(ground Ground, veg Vegetation, burning bool) uint8 {
	value := uint8(ground) & displayGroundMask
	if veg != VegetationNone {
		value |= (uint8(veg) << displayVegetationShift) & displayVegetationMask
	}
	if burning {
		value |= displayBurningBit
	}
	return value
}

func (w *World) rebuildDisplay() {
	total := len(w.display)
	if total == 0 {
		w.heatField = w.heatField[:0]
		return
	}
	if len(w.heatField) != total {
		w.heatField = make([]float32, total)
	}

	burnSpan := w.cfg.Params.BurnTTL
	if burnSpan <= 0 {
		burnSpan = 1
	}
	invBurnSpan := 1.0 / float64(burnSpan)

	for i := 0; i < total; i++ {
		var ground Ground
		if i < len(w.groundCurr) {
			ground = w.groundCurr[i]
		}
		var veg Vegetation
		if i < len(w.vegCurr) {
			veg = w.vegCurr[i]
		}
		burning := false
		if i < len(w.burnTTL) {
			burning = w.burnTTL[i] > 0
		}
		w.display[i] = encodeDisplayValue(ground, veg, burning)

		lavaTemp := float64(0)
		if i < len(w.lavaTemp) {
			lavaTemp = float64(w.lavaTemp[i])
		}
		lavaHeight := uint8(0)
		if i < len(w.lavaHeight) {
			lavaHeight = w.lavaHeight[i]
		}
		burnTTL := uint8(0)
		if i < len(w.burnTTL) {
			burnTTL = w.burnTTL[i]
		}
		w.heatField[i] = float32(computeHeatIntensity(lavaTemp, lavaHeight, burnTTL, invBurnSpan))
	}
}

func computeHeatIntensity(lavaTemp float64, lavaHeight uint8, burnTTL uint8, invBurnSpan float64) float64 {
	lavaComponent := clampFloat(lavaTemp, 0, 1)
	if lavaHeight > 0 {
		heightNorm := float64(lavaHeight) / float64(lavaMaxHeight)
		// Blend temperature and column thickness so tall flows stay vivid even as they cool.
		blended := lavaComponent*0.65 + heightNorm*0.35
		if blended < heightNorm {
			blended = heightNorm
		}
		lavaComponent = clampFloat(blended, 0, 1)
	} else {
		lavaComponent = clampFloat(lavaComponent*0.85, 0, 1)
	}

	fireComponent := 0.0
	if burnTTL > 0 {
		ratio := clampFloat(float64(burnTTL)*invBurnSpan, 0, 1)
		fireComponent = 0.55 + 0.45*math.Sqrt(ratio)
	}

	heat := math.Max(lavaComponent, fireComponent)
	if heat < 1e-3 {
		return 0
	}
	return clampFloat(heat, 0, 1)
}
