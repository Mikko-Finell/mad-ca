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

type windFieldProvider interface {
	WindVectorAt(x, y float64) (float64, float64)
}

type elevationFieldProvider interface {
	ElevationField() []int16
}

// Overlay draws optional debugging visuals on top of the base simulation.
type Overlay struct {
	sim         core.Sim
	scale       int
	showRain    bool
	showVolcano bool
	showWind    bool
	showElev    bool
	maskImg     *ebiten.Image
	maskBuf     []byte

	elevationImg *ebiten.Image
	elevationBuf []byte

	pixel          *ebiten.Image
	windSamples    []windSample
	windCacheW     int
	windCacheH     int
	windCacheScale int
	windPixelSpan  float64
}

type windSample struct {
	cx float64
	cy float64
	sx float64
	sy float64
}

// NewOverlay constructs a new overlay instance.
func NewOverlay(sim core.Sim, scale int) *Overlay {
	o := &Overlay{sim: sim, scale: scale}
	o.pixel = ebiten.NewImage(1, 1)
	o.pixel.Fill(color.White)
	return o
}

// Update allows the overlay to update internal state.
func (o *Overlay) Update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		o.showRain = !o.showRain
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		o.showVolcano = !o.showVolcano
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit3) {
		o.showWind = !o.showWind
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit4) {
		o.showElev = !o.showElev
	}
}

// Draw renders the overlay onto the provided screen.
func (o *Overlay) Draw(screen *ebiten.Image) {
	size := o.sim.Size()
	if size.W <= 0 || size.H <= 0 {
		return
	}
	scale := o.scale
	if scale <= 0 {
		scale = 1
	}

	if o.showWind {
		if provider, ok := o.sim.(windFieldProvider); ok {
			o.drawWindField(screen, provider, size, scale)
		}
	}

	if o.showElev {
		if provider, ok := o.sim.(elevationFieldProvider); ok {
			o.drawElevation(screen, provider.ElevationField(), size, scale)
		}
	}

	if provider, ok := o.sim.(maskProvider); ok {
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
}

func (o *Overlay) drawWindField(screen *ebiten.Image, provider windFieldProvider, size core.Size, scale int) {
	if o.pixel == nil {
		return
	}
	if !o.ensureWindSamples(size, scale) {
		return
	}

	const (
		calmThreshold    = 0.05
		maxSpeedEstimate = 1.1
		headAngle        = math.Pi / 6
		calmDotScale     = 0.18
		minThickness     = 0.65
		maxThickness     = 1.05
	)

	baseSpan := o.windPixelSpan
	if baseSpan <= 0 {
		baseSpan = float64(scale) * 4
	}
	minLength := baseSpan * 0.35
	maxLength := baseSpan * 0.7
	if maxLength < minLength {
		maxLength = minLength
	}

	calmDotSize := baseSpan * calmDotScale
	if calmDotSize < float64(scale)*0.75 {
		calmDotSize = float64(scale) * 0.75
	}

	for _, sample := range o.windSamples {
		vx, vy := provider.WindVectorAt(sample.cx, sample.cy)
		speed := math.Hypot(vx, vy)
		if speed < calmThreshold {
			o.drawPoint(screen, sample.sx, sample.sy, calmDotSize, color.RGBA{R: 90, G: 130, B: 170, A: 120})
			continue
		}

		nx := vx / speed
		ny := vy / speed
		normalized := clamp01(speed / maxSpeedEstimate)
		length := minLength + (maxLength-minLength)*math.Sqrt(normalized)
		headLength := math.Min(length*0.3, float64(scale)*4.5)
		tailLength := length * 0.4
		tipX := sample.sx + nx*(length-tailLength)
		tipY := sample.sy + ny*(length-tailLength)
		tailX := sample.sx - nx*tailLength
		tailY := sample.sy - ny*tailLength
		bodyEndX := tipX - nx*headLength
		bodyEndY := tipY - ny*headLength

		thickness := float64(scale) * (minThickness + (maxThickness-minThickness)*normalized)
		if thickness < 1 {
			thickness = 1
		}

		col := interpolateColor(normalized)
		o.drawLine(screen, tailX, tailY, bodyEndX, bodyEndY, thickness, col)

		angle := math.Atan2(ny, nx)
		leftX := tipX - math.Cos(angle+headAngle)*headLength
		leftY := tipY - math.Sin(angle+headAngle)*headLength
		rightX := tipX - math.Cos(angle-headAngle)*headLength
		rightY := tipY - math.Sin(angle-headAngle)*headLength
		o.drawLine(screen, tipX, tipY, leftX, leftY, thickness*0.85, col)
		o.drawLine(screen, tipX, tipY, rightX, rightY, thickness*0.85, col)
	}
}

func (o *Overlay) ensureWindSamples(size core.Size, scale int) bool {
	if size.W <= 0 || size.H <= 0 {
		return false
	}
	if scale <= 0 {
		scale = 1
	}
	if o.windCacheW == size.W && o.windCacheH == size.H && o.windCacheScale == scale && len(o.windSamples) > 0 {
		return true
	}

	const (
		targetSamples = 360.0
		minSpacing    = 6
		maxSpacing    = 20
	)

	area := float64(size.W * size.H)
	spacing := int(math.Sqrt(area / targetSamples))
	if spacing < minSpacing {
		spacing = minSpacing
	}
	if spacing > maxSpacing {
		spacing = maxSpacing
	}
	if spacing <= 0 {
		spacing = minSpacing
	}

	countX := (size.W + spacing - 1) / spacing
	if countX <= 0 {
		countX = 1
	}
	countY := (size.H + spacing - 1) / spacing
	if countY <= 0 {
		countY = 1
	}

	totalSpanX := (countX - 1) * spacing
	totalSpanY := (countY - 1) * spacing
	startX := 0
	startY := 0
	if size.W > 0 {
		startX = (size.W - 1 - totalSpanX) / 2
		if startX < 0 {
			startX = 0
		}
	}
	if size.H > 0 {
		startY = (size.H - 1 - totalSpanY) / 2
		if startY < 0 {
			startY = 0
		}
	}

	o.windSamples = o.windSamples[:0]
	for yi := 0; yi < countY; yi++ {
		cellY := startY + yi*spacing
		if cellY >= size.H {
			cellY = size.H - 1
		}
		cy := float64(cellY) + 0.5
		for xi := 0; xi < countX; xi++ {
			cellX := startX + xi*spacing
			if cellX >= size.W {
				cellX = size.W - 1
			}
			cx := float64(cellX) + 0.5
			sx := cx * float64(scale)
			sy := cy * float64(scale)
			o.windSamples = append(o.windSamples, windSample{cx: cx, cy: cy, sx: sx, sy: sy})
		}
	}

	o.windCacheW = size.W
	o.windCacheH = size.H
	o.windCacheScale = scale
	o.windPixelSpan = float64(spacing) * float64(scale)
	return len(o.windSamples) > 0
}

func (o *Overlay) drawPoint(screen *ebiten.Image, x, y, size float64, col color.RGBA) {
	if o.pixel == nil || size <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(size, size)
	op.GeoM.Translate(x-size*0.5, y-size*0.5)
	op.ColorM.Scale(float64(col.R)/255.0, float64(col.G)/255.0, float64(col.B)/255.0, float64(col.A)/255.0)
	screen.DrawImage(o.pixel, op)
}

func (o *Overlay) drawLine(screen *ebiten.Image, x1, y1, x2, y2, thickness float64, col color.RGBA) {
	if o.pixel == nil || thickness <= 0 {
		return
	}
	dx := x2 - x1
	dy := y2 - y1
	length := math.Hypot(dx, dy)
	if length <= 1e-4 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(length, thickness)
	op.GeoM.Translate(0, -thickness/2)
	op.GeoM.Rotate(math.Atan2(dy, dx))
	op.GeoM.Translate(x1, y1)
	op.ColorM.Scale(float64(col.R)/255.0, float64(col.G)/255.0, float64(col.B)/255.0, float64(col.A)/255.0)
	screen.DrawImage(o.pixel, op)
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

func (o *Overlay) drawElevation(screen *ebiten.Image, field []int16, size core.Size, scale int) {
	total := size.W * size.H
	if len(field) != total || total == 0 {
		return
	}
	if o.elevationImg == nil || o.elevationImg.Bounds().Dx() != size.W || o.elevationImg.Bounds().Dy() != size.H {
		o.elevationImg = ebiten.NewImage(size.W, size.H)
		o.elevationBuf = make([]byte, 4*total)
	} else if len(o.elevationBuf) != 4*total {
		o.elevationBuf = make([]byte, 4*total)
	}

	minVal := field[0]
	maxVal := field[0]
	for _, v := range field {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	rangeVal := float64(maxVal - minVal)
	if rangeVal == 0 {
		rangeVal = 1
	}
	slopeScale := 0.0
	if maxVal > minVal {
		slopeScale = 1.0 / rangeVal
	}

	for y := 0; y < size.H; y++ {
		for x := 0; x < size.W; x++ {
			idx := y*size.W + x
			base := idx * 4
			normalized := clamp01(float64(field[idx]-minVal) / rangeVal)
			col := elevationColor(normalized)

			alpha := float64(col.A)
			if slopeScale > 0 {
				baseVal := int(field[idx])
				maxDiff := 0
				if x > 0 {
					diff := absInt(baseVal - int(field[idx-1]))
					if diff > maxDiff {
						maxDiff = diff
					}
				}
				if x+1 < size.W {
					diff := absInt(baseVal - int(field[idx+1]))
					if diff > maxDiff {
						maxDiff = diff
					}
				}
				if y > 0 {
					diff := absInt(baseVal - int(field[idx-size.W]))
					if diff > maxDiff {
						maxDiff = diff
					}
				}
				if y+1 < size.H {
					diff := absInt(baseVal - int(field[idx+size.W]))
					if diff > maxDiff {
						maxDiff = diff
					}
				}
				slope := clamp01(float64(maxDiff) * slopeScale)
				alpha *= 0.55 + 0.45*slope
			}

			o.elevationBuf[base+0] = col.R
			o.elevationBuf[base+1] = col.G
			o.elevationBuf[base+2] = col.B
			o.elevationBuf[base+3] = uint8(math.Round(clamp(alpha, 0, 255)))
		}
	}

	o.elevationImg.ReplacePixels(o.elevationBuf)
	op := &ebiten.DrawImageOptions{}
	if scale <= 0 {
		scale = 1
	}
	op.GeoM.Scale(float64(scale), float64(scale))
	screen.DrawImage(o.elevationImg, op)
}

func interpolateColor(t float64) color.RGBA {
	t = clamp01(t)
	r := uint8(math.Round(80 + 70*t))
	g := uint8(math.Round(170 + 70*t))
	b := uint8(math.Round(230 + 20*t))
	a := uint8(math.Round(150 + 90*t))
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
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

func elevationColor(t float64) color.RGBA {
	t = clamp01(t)
	stops := []struct {
		t   float64
		col color.RGBA
	}{
		{0.0, color.RGBA{R: 40, G: 60, B: 120, A: 150}},
		{0.25, color.RGBA{R: 70, G: 105, B: 160, A: 165}},
		{0.5, color.RGBA{R: 90, G: 150, B: 100, A: 185}},
		{0.75, color.RGBA{R: 190, G: 160, B: 80, A: 205}},
		{1.0, color.RGBA{R: 240, G: 235, B: 215, A: 215}},
	}
	for i := 1; i < len(stops); i++ {
		curr := stops[i]
		if t <= curr.t {
			prev := stops[i-1]
			span := curr.t - prev.t
			var local float64
			if span > 0 {
				local = (t - prev.t) / span
			}
			return lerpRGBA(prev.col, curr.col, clamp01(local))
		}
	}
	return stops[len(stops)-1].col
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	t = clamp01(t)
	return color.RGBA{
		R: lerpComponent(a.R, b.R, t),
		G: lerpComponent(a.G, b.G, t),
		B: lerpComponent(a.B, b.B, t),
		A: lerpComponent(a.A, b.A, t),
	}
}

func lerpComponent(a, b uint8, t float64) uint8 {
	return uint8(math.Round(float64(a) + (float64(b)-float64(a))*t))
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
