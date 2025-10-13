package ecology

import "image/color"

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
	}
}
