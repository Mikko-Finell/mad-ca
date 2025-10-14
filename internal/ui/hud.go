//go:build ebiten

package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"mad-ca/internal/core"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
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

	controls      []hudControlState
	intSetter     core.IntParameterSetter
	floatSetter   core.FloatParameterSetter
	panelOffsetX  int
	title         string
	scrollOffset  int
	contentHeight int

	pixel *ebiten.Image
}

// NewHUD constructs a HUD for the provided simulation and panel width.
func NewHUD(sim core.Sim, width int) *HUD {
	if width < 0 {
		width = 0
	}
	h := &HUD{sim: sim, width: width}
	if width > 0 {
		h.pixel = ebiten.NewImage(1, 1)
		h.pixel.Fill(color.White)
	}
	h.title = buildTitle(sim)
	if provider, ok := sim.(core.ParameterControlsProvider); ok {
		controls := provider.ParameterControls()
		h.controls = make([]hudControlState, len(controls))
		for i, ctrl := range controls {
			h.controls[i] = hudControlState{control: ctrl, value: "--"}
		}
		h.layoutControls()
	}
	if setter, ok := sim.(core.IntParameterSetter); ok {
		h.intSetter = setter
	}
	if setter, ok := sim.(core.FloatParameterSetter); ok {
		h.floatSetter = setter
	}
	return h
}

// Update refreshes the cached parameter snapshot from the simulation and handles
// HUD interactions.
func (h *HUD) Update(panelOffsetX int) {
	if h == nil {
		return
	}
	h.panelOffsetX = panelOffsetX
	provider, ok := h.sim.(parameterProvider)
	if !ok {
		h.snapshot = core.ParameterSnapshot{}
		return
	}
	h.snapshot = provider.Parameters()
	h.refreshControlValues()
	h.handleInput()
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
	h.panel.Fill(color.RGBA{R: 16, G: 16, B: 20, A: 255})
	h.clampScroll()
	h.drawControls()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(offsetX), 0)
	screen.DrawImage(h.panel, op)
}

func buildTitle(sim core.Sim) string {
	if sim == nil {
		return "Controls"
	}
	name := sim.Name()
	if name == "" {
		return "Controls"
	}
	if strings.EqualFold(name, "ecology") {
		return ""
	}
	return fmt.Sprintf("%s Controls", strings.Title(name))
}

func (h *HUD) refreshControlValues() {
	if len(h.controls) == 0 {
		return
	}
	paramMap := map[string]core.Parameter{}
	for _, group := range h.snapshot.Groups {
		for _, param := range group.Params {
			paramMap[param.Key] = param
		}
	}
	for i := range h.controls {
		state := &h.controls[i]
		param, ok := paramMap[state.control.Key]
		if !ok {
			state.hasValue = false
			state.value = "--"
			continue
		}
		switch state.control.Type {
		case core.ParamTypeInt:
			parsed, err := strconv.Atoi(param.Value)
			if err != nil {
				state.hasValue = false
				state.value = "--"
				continue
			}
			state.intValue = parsed
			state.floatValue = float64(parsed)
			state.value = strconv.Itoa(parsed)
			state.hasValue = true
		case core.ParamTypeFloat:
			parsed, err := strconv.ParseFloat(param.Value, 64)
			if err != nil {
				state.hasValue = false
				state.value = "--"
				continue
			}
			value := parsed
			if isChanceControl(state.control) {
				value = parsed * 100
			}
			state.floatValue = value
			state.value = h.formatFloat(state, value)
			state.hasValue = true
		default:
			state.hasValue = false
			state.value = "--"
		}
	}
}

func (h *HUD) handleInput() {
	if len(h.controls) == 0 {
		return
	}
	mx, my := ebiten.CursorPosition()
	withinPanel := mx >= h.panelOffsetX && mx < h.panelOffsetX+h.width
	if withinPanel {
		_, wy := ebiten.Wheel()
		if wy != 0 {
			delta := int(math.Round(wy * float64(scrollStep)))
			if delta != 0 {
				h.scrollBy(-delta)
			}
		}
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	if !withinPanel {
		return
	}
	px := mx - h.panelOffsetX
	for i := range h.controls {
		state := &h.controls[i]
		if !state.hasValue {
			continue
		}
		minusRect := offsetRect(state.minusRect, 0, -h.scrollOffset)
		plusRect := offsetRect(state.plusRect, 0, -h.scrollOffset)
		if pointInRect(px, my, minusRect) {
			h.applyAdjustment(state, -1)
			return
		}
		if pointInRect(px, my, plusRect) {
			h.applyAdjustment(state, 1)
			return
		}
	}
}

func (h *HUD) applyAdjustment(state *hudControlState, direction int) {
	if state == nil || direction == 0 {
		return
	}
	switch state.control.Type {
	case core.ParamTypeInt:
		if h.intSetter == nil {
			return
		}
		step := int(math.Round(state.control.Step))
		if step <= 0 {
			step = 1
		}
		target := state.intValue + direction*step
		if state.control.HasMin {
			min := int(math.Round(state.control.Min))
			if target < min {
				target = min
			}
		}
		if state.control.HasMax {
			max := int(math.Round(state.control.Max))
			if target > max {
				target = max
			}
		}
		if target == state.intValue {
			return
		}
		if h.intSetter.SetIntParameter(state.control.Key, target) {
			state.intValue = target
			state.floatValue = float64(target)
			state.value = strconv.Itoa(target)
		}
	case core.ParamTypeFloat:
		if h.floatSetter == nil {
			return
		}
		step := h.floatStep(state)
		target := state.floatValue + float64(direction)*step
		if state.control.HasMin && target < state.control.Min {
			target = state.control.Min
		}
		if state.control.HasMax && target > state.control.Max {
			target = state.control.Max
		}
		if math.Abs(target-state.floatValue) < 1e-9 {
			return
		}
		if h.floatSetter.SetFloatParameter(state.control.Key, target) {
			state.floatValue = target
			state.value = h.formatFloat(state, target)
		}
	}
}

func (h *HUD) drawControls() {
	if h.panel == nil {
		return
	}
	face := basicfont.Face7x13
	headerY := panelPadding + headerBaseline
	text.Draw(h.panel, h.title, face, panelPadding, headerY, color.RGBA{R: 200, G: 200, B: 210, A: 255})
	if len(h.controls) == 0 {
		infoY := h.controlsTop() + emptyControlsOffset
		text.Draw(h.panel, "No adjustable parameters", face, panelPadding, infoY, color.RGBA{R: 160, G: 160, B: 170, A: 255})
		return
	}
	panelHeight := h.lastHeight
	if panelHeight == 0 && h.panel != nil {
		panelHeight = h.panel.Bounds().Dy()
	}
	controlsStart := h.controlsTop()
	for i := range h.controls {
		state := &h.controls[i]
		top := state.top - h.scrollOffset
		if top+lineHeight < controlsStart {
			continue
		}
		if panelHeight > 0 && top >= panelHeight {
			continue
		}
		labelY := top + labelBaseline
		text.Draw(h.panel, state.control.Label, face, panelPadding, labelY, color.RGBA{R: 220, G: 220, B: 230, A: 255})
		valueColor := color.RGBA{R: 220, G: 220, B: 230, A: 255}
		if !state.hasValue {
			valueColor = color.RGBA{R: 160, G: 160, B: 170, A: 255}
		}
		value := state.value
		bounds := text.BoundString(face, value)
		valueWidth := bounds.Dx()
		minusRect := offsetRect(state.minusRect, 0, -h.scrollOffset)
		plusRect := offsetRect(state.plusRect, 0, -h.scrollOffset)
		valueX := minusRect.Min.X - buttonGap - valueWidth
		valueY := top + valueBaseline
		text.Draw(h.panel, value, face, valueX, valueY, valueColor)

		minusEnabled := state.hasValue && h.canAdjust(state, -1)
		plusEnabled := state.hasValue && h.canAdjust(state, 1)
		h.drawButton(minusRect, "-", minusEnabled)
		h.drawButton(plusRect, "+", plusEnabled)
	}
}

func (h *HUD) canAdjust(state *hudControlState, direction int) bool {
	if state == nil || direction == 0 {
		return false
	}
	switch state.control.Type {
	case core.ParamTypeInt:
		if h.intSetter == nil {
			return false
		}
		step := int(math.Round(state.control.Step))
		if step <= 0 {
			step = 1
		}
		target := state.intValue + direction*step
		if state.control.HasMin {
			min := int(math.Round(state.control.Min))
			if direction < 0 && target < min {
				return false
			}
		}
		if state.control.HasMax {
			max := int(math.Round(state.control.Max))
			if direction > 0 && target > max {
				return false
			}
		}
		return true
	case core.ParamTypeFloat:
		if h.floatSetter == nil {
			return false
		}
		step := h.floatStep(state)
		target := state.floatValue + float64(direction)*step
		if state.control.HasMin && direction < 0 && target < state.control.Min {
			return false
		}
		if state.control.HasMax && direction > 0 && target > state.control.Max {
			return false
		}
		return true
	default:
		return false
	}
}

func (h *HUD) drawButton(rect image.Rectangle, label string, enabled bool) {
	if h.pixel == nil {
		return
	}
	bg := color.RGBA{R: 54, G: 56, B: 64, A: 255}
	fg := color.RGBA{R: 230, G: 230, B: 240, A: 255}
	if !enabled {
		bg = color.RGBA{R: 32, G: 34, B: 40, A: 255}
		fg = color.RGBA{R: 120, G: 120, B: 130, A: 255}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(rect.Dx()), float64(rect.Dy()))
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	op.ColorM.Scale(float64(bg.R)/255.0, float64(bg.G)/255.0, float64(bg.B)/255.0, float64(bg.A)/255.0)
	h.panel.DrawImage(h.pixel, op)

	face := basicfont.Face7x13
	bounds := text.BoundString(face, label)
	textWidth := bounds.Dx()
	textHeight := bounds.Dy()
	x := rect.Min.X + (rect.Dx()-textWidth)/2
	y := rect.Min.Y + (rect.Dy()-textHeight)/2 + textHeight
	text.Draw(h.panel, label, face, x, y, fg)
}

func (h *HUD) controlsTop() int {
	if strings.TrimSpace(h.title) == "" {
		return panelPadding
	}
	return panelPadding + headerBaseline + headerGap
}

func (h *HUD) layoutControls() {
	controlsStart := h.controlsTop()
	if len(h.controls) == 0 || h.width <= 0 {
		h.contentHeight = controlsStart + panelPadding
		h.clampScroll()
		return
	}
	contentBottom := controlsStart
	for i := range h.controls {
		top := controlsStart + i*lineHeight
		buttonY := top + buttonRowTop
		plusRect := image.Rect(h.width-panelPadding-buttonSize, buttonY, h.width-panelPadding, buttonY+buttonSize)
		minusRect := image.Rect(plusRect.Min.X-buttonGap-buttonSize, buttonY, plusRect.Min.X-buttonGap, buttonY+buttonSize)
		h.controls[i].top = top
		h.controls[i].minusRect = minusRect
		h.controls[i].plusRect = plusRect
		if bottom := top + lineHeight; bottom > contentBottom {
			contentBottom = bottom
		}
	}
	h.contentHeight = contentBottom + panelPadding
	h.clampScroll()
}

func (h *HUD) formatFloat(state *hudControlState, value float64) string {
	if state == nil {
		return strconv.FormatFloat(value, 'f', 2, 64)
	}
	step := h.floatStep(state)
	if step <= 0 {
		step = 0.01
	}
	precision := 0
	switch {
	case step >= 1:
		precision = 0
	case step >= 0.1:
		precision = 1
	case step >= 0.01:
		precision = 2
	case step >= 0.001:
		precision = 3
	case step >= 0.0001:
		precision = 4
	default:
		precision = 6
	}
	return strconv.FormatFloat(value, 'f', precision, 64)
}

func (h *HUD) floatStep(state *hudControlState) float64 {
	if state == nil {
		return 0.01
	}
	ctrl := state.control
	if ctrl.Step > 0 {
		return ctrl.Step
	}
	if ctrl.HasMin && ctrl.HasMax {
		span := ctrl.Max - ctrl.Min
		if span > 0 {
			step := span / 100
			if step > 0 {
				return step
			}
		}
	}
	value := math.Abs(state.floatValue)
	if value > 0 {
		step := value / 10
		if step > 0 {
			return step
		}
	}
	if ctrl.HasMin {
		base := math.Abs(ctrl.Min)
		if base > 0 {
			step := base / 10
			if step > 0 {
				return step
			}
		}
	}
	if ctrl.HasMax && !math.IsInf(ctrl.Max, 1) {
		base := math.Abs(ctrl.Max)
		if base > 0 {
			step := base / 100
			if step > 0 {
				return step
			}
		}
	}
	return 0.01
}

func isChanceControl(ctrl core.ParameterControl) bool {
	if ctrl.Key == "" {
		return false
	}
	return strings.Contains(ctrl.Key, "chance")
}

func pointInRect(x, y int, rect image.Rectangle) bool {
	return x >= rect.Min.X && x < rect.Max.X && y >= rect.Min.Y && y < rect.Max.Y
}

type hudControlState struct {
	control core.ParameterControl
	value   string

	intValue   int
	floatValue float64
	hasValue   bool

	top       int
	minusRect image.Rectangle
	plusRect  image.Rectangle
}

const (
	panelPadding        = 12
	lineHeight          = 50
	buttonSize          = 16
	buttonGap           = 6
	headerBaseline      = 18
	labelBaseline       = 24
	valueBaseline       = 38
	buttonRowTop        = 28
	headerGap           = 14
	emptyControlsOffset = 54
	scrollStep          = 24
)

func (h *HUD) scrollBy(delta int) {
	if delta == 0 {
		return
	}
	h.scrollOffset += delta
	h.clampScroll()
}

func (h *HUD) clampScroll() {
	panelHeight := h.lastHeight
	if panelHeight <= 0 && h.panel != nil {
		panelHeight = h.panel.Bounds().Dy()
	}
	maxScroll := h.contentHeight - panelHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if h.scrollOffset < 0 {
		h.scrollOffset = 0
	}
	if h.scrollOffset > maxScroll {
		h.scrollOffset = maxScroll
	}
}

func offsetRect(rect image.Rectangle, dx, dy int) image.Rectangle {
	if dx == 0 && dy == 0 {
		return rect
	}
	return image.Rect(rect.Min.X+dx, rect.Min.Y+dy, rect.Max.X+dx, rect.Max.Y+dy)
}
