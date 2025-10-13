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

	controls     []hudControlState
	intSetter    core.IntParameterSetter
	floatSetter  core.FloatParameterSetter
	panelOffsetX int
	title        string

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
			state.floatValue = parsed
			state.value = h.formatFloat(state.control, parsed)
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
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	mx, my := ebiten.CursorPosition()
	if mx < h.panelOffsetX {
		return
	}
	px := mx - h.panelOffsetX
	for i := range h.controls {
		state := &h.controls[i]
		if !state.hasValue {
			continue
		}
		if pointInRect(px, my, state.minusRect) {
			h.applyAdjustment(state, -1)
			return
		}
		if pointInRect(px, my, state.plusRect) {
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
		step := state.control.Step
		if step <= 0 {
			step = 0.05
		}
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
			state.value = h.formatFloat(state.control, target)
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
		infoY := headerY + infoSpacing
		text.Draw(h.panel, "No adjustable parameters", face, panelPadding, infoY, color.RGBA{R: 160, G: 160, B: 170, A: 255})
		return
	}
	for i := range h.controls {
		state := &h.controls[i]
		top := state.top
		labelY := top + labelBaseline
		text.Draw(h.panel, state.control.Label, face, panelPadding, labelY, color.RGBA{R: 220, G: 220, B: 230, A: 255})
		valueColor := color.RGBA{R: 220, G: 220, B: 230, A: 255}
		if !state.hasValue {
			valueColor = color.RGBA{R: 160, G: 160, B: 170, A: 255}
		}
		value := state.value
		bounds := text.BoundString(face, value)
		valueWidth := bounds.Dx()
		valueX := state.minusRect.Min.X - buttonGap - valueWidth
		valueY := top + labelBaseline
		text.Draw(h.panel, value, face, valueX, valueY, valueColor)

		minusEnabled := state.hasValue && h.canAdjust(state, -1)
		plusEnabled := state.hasValue && h.canAdjust(state, 1)
		h.drawButton(state.minusRect, "-", minusEnabled)
		h.drawButton(state.plusRect, "+", plusEnabled)
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
		step := state.control.Step
		if step <= 0 {
			step = 0.05
		}
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

func (h *HUD) layoutControls() {
	if len(h.controls) == 0 || h.width <= 0 {
		return
	}
	for i := range h.controls {
		top := controlsTop + i*lineHeight
		buttonY := top + (lineHeight-buttonSize)/2
		plusRect := image.Rect(h.width-panelPadding-buttonSize, buttonY, h.width-panelPadding, buttonY+buttonSize)
		minusRect := image.Rect(plusRect.Min.X-buttonGap-buttonSize, buttonY, plusRect.Min.X-buttonGap, buttonY+buttonSize)
		h.controls[i].top = top
		h.controls[i].minusRect = minusRect
		h.controls[i].plusRect = plusRect
	}
}

func (h *HUD) formatFloat(ctrl core.ParameterControl, value float64) string {
	step := ctrl.Step
	if step <= 0 {
		step = 0.05
	}
	precision := 2
	switch {
	case step < 0.001:
		precision = 4
	case step < 0.01:
		precision = 3
	case step < 0.1:
		precision = 2
	default:
		precision = 1
	}
	return strconv.FormatFloat(value, 'f', precision, 64)
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
	panelPadding   = 12
	lineHeight     = 36
	buttonSize     = 24
	buttonGap      = 6
	headerBaseline = 18
	labelBaseline  = 24
	infoSpacing    = 36
	controlsTop    = panelPadding + headerBaseline + 14
)
