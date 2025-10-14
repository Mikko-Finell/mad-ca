package ecology

import (
	"math"
	"math/rand"

	"mad-ca/internal/core"
)

// Ground enumerates the terrain layer values.
type Ground uint8

// Vegetation enumerates the vegetation layer values.
type Vegetation uint8

const (
	GroundDirt Ground = iota
	GroundRock
	GroundMountain
	GroundLava
)

const (
	VegetationNone Vegetation = iota
	VegetationGrass
	VegetationShrub
	VegetationTree
)

// World stores all phase-one state for the ecology simulation.
type World struct {
	cfg Config

	w, h int

	groundCurr     []Ground
	groundNext     []Ground
	vegCurr        []Vegetation
	vegNext        []Vegetation
	lavaHeight     []uint8
	lavaHeightNext []uint8
	lavaTemp       []float32
	lavaTempNext   []float32
	lavaDir        []int8
	lavaDirNext    []int8
	lavaTip        []bool
	lavaTipNext    []bool
	lavaForce      []bool
	lavaForceNext  []bool
	lavaChannel    []float32
	lavaElevation  []int16
	lavaNoise      []int8
	burnTTL        []uint8
	burnNext       []uint8
	rainCurr       []float32
	rainNext       []float32
	rainScratch    []float32
	volCurr        []float32
	volNext        []float32
	tectonic       []float32
	display        []uint8

	rng *rand.Rand

	metrics VegetationMetrics

	rainRegions          []rainRegion
	volcanoRegions       []volcanoProtoRegion
	expiredVolcanoProtos []volcanoProtoRegion
	lavaVents            []lavaVent
	lavaReservoir        lavaReservoir

	windPhase float64

	lavaTipQueue      []int
	lavaFailedTips    []int
	lavaAdvancedCells []int
}

type lavaReservoir struct {
	cells []int
	ticks int
}

func (r *lavaReservoir) reset() {
	if r == nil {
		return
	}
	r.cells = r.cells[:0]
	r.ticks = 0
}

func (r *lavaReservoir) assign(cells []int, ticks int) {
	if r == nil {
		return
	}
	r.cells = append(r.cells[:0], cells...)
	if ticks < 0 {
		ticks = 0
	}
	r.ticks = ticks
}

// VegetationMetrics captures aggregate vegetation telemetry for the current tick.
type VegetationMetrics struct {
	GrassTiles int
	ShrubTiles int
	TreeTiles  int

	TotalVegetated int

	// ClusterHistogram stores the count of Moore-connected components by size.
	// Index represents the component size; index 0 is unused.
	ClusterHistogram []int
}

// EnvironmentMetrics captures ground, fire, and rain telemetry for the current tick.
type EnvironmentMetrics struct {
	DirtTiles     int
	RockTiles     int
	MountainTiles int
	LavaTiles     int

	BurningTiles int

	RainCoverage      int
	ActiveRainRegions int
	RainMean          float64
	RainMax           float64

	TotalTiles int
}

type volcanoProtoRegion struct {
	cx, cy   float64
	radius   float64
	strength float64
	ttl      int
	noise    int64
}

type rainRegion struct {
	cx, cy             float64
	radiusX            float64
	radiusY            float64
	strength           float64
	baseStrength       float64
	strengthVariation  float64
	targetBaseStrength float64
	targetRadiusX      float64
	targetRadiusY      float64
	ttl                int
	maxTTL             int
	age                int
	vx, vy             float64
	threshold          float64
	falloff            float64
	noiseScale         float64
	noiseStretchX      float64
	noiseStretchY      float64
	noiseSeed          int64
	noiseOffsetX       float64
	noiseOffsetY       float64
	angle              float64
	preset             rainPreset
	mergeTicks         int
}

type lavaVent struct {
	idx    int
	ttl    int
	dir    int8
	outIdx int
}

type lavaDirection struct {
	dx int
	dy int
	ux float64
	uy float64
}

type lavaCandidate struct {
	idx   int
	dir   int8
	score float64
}

const (
	lavaMaxHeight             = 7
	lavaFluxPerTick           = 1
	lavaFlowThreshold         = 0.9
	lavaSplitThresholdDrop    = 0.15
	lavaSplitMinHeight        = 3
	lavaSplitChance           = 0.25
	lavaOverflowHeight        = 4
	lavaBaseSpeed             = 0.9
	lavaSpeedAlpha            = 0.3
	lavaCoolBase              = 0.02
	lavaCoolEdge              = 0.03
	lavaCoolRain              = 0.08
	lavaCoolThick             = 0.02
	lavaCoolFlowBonus         = 0.02
	lavaCrustThreshold        = 0.15
	lavaTipTemperatureMin     = 0.12
	lavaReheatCap             = 0.35
	lavaSlopeWeight           = 1.0
	lavaAlignWeight           = 0.6
	lavaChannelWeight         = 0.8
	lavaRainWeight            = 0.5
	lavaWallWeight            = 2.0
	lavaChannelGrow           = 0.15
	lavaChannelDecay          = 0.005
	lavaReservoirTargetHeight = 3
	lavaReservoirMinTemp      = 0.75
)

var lavaDirections = [...]lavaDirection{
	{dx: 1, dy: 0, ux: 1, uy: 0},
	{dx: 1, dy: 1, ux: 1 / math.Sqrt2, uy: 1 / math.Sqrt2},
	{dx: 0, dy: 1, ux: 0, uy: 1},
	{dx: -1, dy: 1, ux: -1 / math.Sqrt2, uy: 1 / math.Sqrt2},
	{dx: -1, dy: 0, ux: -1, uy: 0},
	{dx: -1, dy: -1, ux: -1 / math.Sqrt2, uy: -1 / math.Sqrt2},
	{dx: 0, dy: -1, ux: 0, uy: -1},
	{dx: 1, dy: -1, ux: 1 / math.Sqrt2, uy: -1 / math.Sqrt2},
}

func lavaDirFromDelta(dx, dy int) int8 {
	for i, dir := range lavaDirections {
		if dir.dx == dx && dir.dy == dy {
			return int8(i)
		}
	}
	return -1
}

func lavaLeft(dir int8) int8 {
	if dir < 0 {
		return -1
	}
	return int8((int(dir) + len(lavaDirections) - 1) % len(lavaDirections))
}

func lavaRight(dir int8) int8 {
	if dir < 0 {
		return -1
	}
	return int8((int(dir) + 1) % len(lavaDirections))
}

func lavaDot(a, b int8) float64 {
	if a < 0 || b < 0 {
		return 0
	}
	da := lavaDirections[a]
	db := lavaDirections[b]
	return da.ux*db.ux + da.uy*db.uy
}

func lavaSigmoid(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}

func (w *World) setLavaCell(idx int, height int, temp float32, dir int8, tip bool) {
	if idx < 0 || idx >= len(w.groundCurr) {
		return
	}
	if height <= 0 {
		w.groundCurr[idx] = GroundRock
		if idx < len(w.lavaHeight) {
			w.lavaHeight[idx] = 0
		}
		if idx < len(w.lavaHeightNext) {
			w.lavaHeightNext[idx] = 0
		}
		if idx < len(w.lavaTemp) {
			w.lavaTemp[idx] = 0
		}
		if idx < len(w.lavaTempNext) {
			w.lavaTempNext[idx] = 0
		}
		if idx < len(w.lavaDir) {
			w.lavaDir[idx] = -1
		}
		if idx < len(w.lavaDirNext) {
			w.lavaDirNext[idx] = -1
		}
		if idx < len(w.lavaTip) {
			w.lavaTip[idx] = false
		}
		if idx < len(w.lavaTipNext) {
			w.lavaTipNext[idx] = false
		}
		if idx < len(w.lavaForce) {
			w.lavaForce[idx] = false
		}
		if idx < len(w.lavaForceNext) {
			w.lavaForceNext[idx] = false
		}
		return
	}
	if height > lavaMaxHeight {
		height = lavaMaxHeight
	}
	if temp < 0 {
		temp = 0
	}
	if temp > 1 {
		temp = 1
	}
	w.groundCurr[idx] = GroundLava
	if idx < len(w.lavaHeight) {
		w.lavaHeight[idx] = uint8(height)
	}
	if idx < len(w.lavaHeightNext) {
		w.lavaHeightNext[idx] = uint8(height)
	}
	if idx < len(w.lavaTemp) {
		w.lavaTemp[idx] = temp
	}
	if idx < len(w.lavaTempNext) {
		w.lavaTempNext[idx] = temp
	}
	if idx < len(w.lavaDir) {
		w.lavaDir[idx] = dir
	}
	if idx < len(w.lavaDirNext) {
		w.lavaDirNext[idx] = dir
	}
	if idx < len(w.lavaTip) {
		w.lavaTip[idx] = tip
	}
	if idx < len(w.lavaTipNext) {
		w.lavaTipNext[idx] = tip
	}
	overflow := height >= lavaOverflowHeight
	if idx < len(w.lavaForce) {
		w.lavaForce[idx] = overflow
	}
	if idx < len(w.lavaForceNext) {
		w.lavaForceNext[idx] = overflow
	}
	if idx < len(w.vegCurr) {
		w.vegCurr[idx] = VegetationNone
	}
	if idx < len(w.vegNext) {
		w.vegNext[idx] = VegetationNone
	}
	if idx < len(w.burnTTL) {
		w.burnTTL[idx] = 0
	}
	if idx < len(w.burnNext) {
		w.burnNext[idx] = 0
	}
	if idx < len(w.display) {
		w.display[idx] = uint8(GroundLava)
	}
}

func (w *World) buildLavaElevation(region volcanoProtoRegion) {
	total := w.w * w.h
	if total == 0 || len(w.lavaElevation) != total {
		return
	}
	slopeScale := float64(20 + w.rng.Intn(21))
	if slopeScale <= 0 {
		slopeScale = 20
	}
	cx := region.cx
	cy := region.cy
	for y := 0; y < w.h; y++ {
		for x := 0; x < w.w; x++ {
			idx := y*w.w + x
			base := int16(3)
			if idx < len(w.groundCurr) {
				switch w.groundCurr[idx] {
				case GroundRock:
					base = 4
				case GroundMountain:
					base = 6
				}
			}
			noise := int16(0)
			if idx < len(w.lavaNoise) {
				noise = int16(w.lavaNoise[idx])
			}
			dx := (float64(x) + 0.5) - cx
			dy := (float64(y) + 0.5) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			slope := int16(math.Round(dist / slopeScale))
			w.lavaElevation[idx] = base + noise - slope
		}
	}
}

func (w *World) clearLavaField() {
	for idx := range w.groundCurr {
		if w.groundCurr[idx] != GroundLava {
			continue
		}
		w.setLavaCell(idx, 0, 0, -1, false)
		if idx < len(w.display) {
			w.display[idx] = uint8(w.groundCurr[idx])
		}
	}
}

func (w *World) pickDownhill(idx int) (int, int8, bool) {
	if idx < 0 || idx >= len(w.lavaElevation) || w.w == 0 {
		return -1, -1, false
	}
	base := w.lavaElevation[idx]
	x := idx % w.w
	y := idx / w.w
	bestIdx := -1
	bestDir := int8(-1)
	var bestDrop int16 = math.MinInt16
	for dirIdx, dir := range lavaDirections {
		nx := x + dir.dx
		ny := y + dir.dy
		if nx < 0 || nx >= w.w || ny < 0 || ny >= w.h {
			continue
		}
		nIdx := ny*w.w + nx
		if nIdx < 0 || nIdx >= len(w.lavaElevation) {
			continue
		}
		drop := base - w.lavaElevation[nIdx]
		if drop > bestDrop {
			bestDrop = drop
			bestIdx = nIdx
			bestDir = int8(dirIdx)
		}
	}
	if bestIdx < 0 {
		return -1, -1, false
	}
	return bestIdx, bestDir, bestDrop > 0
}

type rainPreset int

const (
	rainPresetPuffy rainPreset = iota
	rainPresetStratus
	rainPresetSquall
)

// ParameterControls exposes the key ecology parameters that should be
// adjustable from the HUD.
func (w *World) ParameterControls() []core.ParameterControl {
	return []core.ParameterControl{
		{
			Key:    "grass_spread_chance",
			Label:  "Grass spread chance",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    100,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "shrub_growth_chance",
			Label:  "Shrub growth chance",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    100,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "fire_spread_chance",
			Label:  "Fire spread chance",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    100,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "fire_rain_spread_dampen",
			Label:  "Rain dampen factor",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    1,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "rain_spawn_chance",
			Label:  "Rain spawn chance",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    100,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "rain_strength_max",
			Label:  "Rain strength max",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    1,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "wind_noise_scale",
			Label:  "Wind noise scale",
			Type:   core.ParamTypeFloat,
			Min:    0,
			HasMin: true,
		},
		{
			Key:    "wind_speed_scale",
			Label:  "Wind speed scale",
			Type:   core.ParamTypeFloat,
			Min:    0,
			HasMin: true,
		},
		{
			Key:    "wind_temporal_scale",
			Label:  "Wind temporal scale",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    0.06,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "lava_spread_chance",
			Label:  "Lava spread chance",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    100,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "volcano_proto_spawn_chance",
			Label:  "Volcano proto spawn chance",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    100,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "volcano_eruption_chance_base",
			Label:  "Volcano eruption chance",
			Type:   core.ParamTypeFloat,
			Min:    0,
			Max:    100,
			HasMin: true,
			HasMax: true,
		},
		{
			Key:    "lava_life_min",
			Label:  "Lava life min",
			Type:   core.ParamTypeInt,
			Step:   1,
			Min:    1,
			HasMin: true,
		},
		{
			Key:    "lava_life_max",
			Label:  "Lava life max",
			Type:   core.ParamTypeInt,
			Step:   1,
			Min:    1,
			HasMin: true,
		},
		{
			Key:    "burn_ttl",
			Label:  "Burn duration",
			Type:   core.ParamTypeInt,
			Step:   1,
			Min:    1,
			HasMin: true,
		},
	}
}

// SetIntParameter allows HUD interactions to update integer ecology parameters.
func (w *World) SetIntParameter(key string, value int) bool {
	if w == nil {
		return false
	}
	switch key {
	case "lava_life_min":
		if value < 1 {
			value = 1
		}
		if value > w.cfg.Params.LavaLifeMax {
			w.cfg.Params.LavaLifeMax = value
		}
		w.cfg.Params.LavaLifeMin = value
		w.clampVentLifetimes()
		return true
	case "lava_life_max":
		if value < w.cfg.Params.LavaLifeMin {
			value = w.cfg.Params.LavaLifeMin
		}
		if value < 1 {
			value = 1
		}
		w.cfg.Params.LavaLifeMax = value
		w.clampVentLifetimes()
		return true
	case "burn_ttl":
		if value < 1 {
			value = 1
		}
		w.cfg.Params.BurnTTL = value
		return true
	default:
		return false
	}
}

func (w *World) clampVentLifetimes() {
	if w == nil {
		return
	}
	if len(w.lavaVents) == 0 {
		return
	}
	minLife := w.cfg.Params.LavaLifeMin
	if minLife < 1 {
		minLife = 1
	}
	maxLife := w.cfg.Params.LavaLifeMax
	if maxLife < minLife {
		maxLife = minLife
	}
	for i := range w.lavaVents {
		ttl := w.lavaVents[i].ttl
		if ttl < minLife {
			ttl = minLife
		}
		if ttl > maxLife {
			ttl = maxLife
		}
		w.lavaVents[i].ttl = ttl
	}
}

// SetFloatParameter allows HUD interactions to update float ecology parameters.
func (w *World) SetFloatParameter(key string, value float64) bool {
	if w == nil {
		return false
	}
	switch key {
	case "grass_spread_chance":
		w.cfg.Params.GrassSpreadChance = percentToProbability(value)
		return true
	case "shrub_growth_chance":
		w.cfg.Params.ShrubGrowthChance = percentToProbability(value)
		return true
	case "fire_spread_chance":
		w.cfg.Params.FireSpreadChance = percentToProbability(value)
		return true
	case "fire_rain_spread_dampen":
		w.cfg.Params.FireRainSpreadDampen = clampFloat(value, 0, 1)
		return true
	case "rain_spawn_chance":
		w.cfg.Params.RainSpawnChance = percentToProbability(value)
		return true
	case "rain_strength_max":
		clamped := clampFloat(value, 0, 1)
		if clamped < w.cfg.Params.RainStrengthMin {
			clamped = w.cfg.Params.RainStrengthMin
		}
		w.cfg.Params.RainStrengthMax = clamped
		return true
	case "wind_noise_scale":
		if value < 0 {
			value = 0
		}
		w.cfg.Params.WindNoiseScale = value
		return true
	case "wind_speed_scale":
		if value < 0 {
			value = 0
		}
		w.cfg.Params.WindSpeedScale = value
		return true
	case "wind_temporal_scale":
		if value < 0 {
			value = 0
		}
		w.cfg.Params.WindTemporalScale = value
		return true
	case "lava_spread_chance":
		w.cfg.Params.LavaSpreadChance = percentToProbability(value)
		return true
	case "volcano_proto_spawn_chance":
		w.cfg.Params.VolcanoProtoSpawnChance = percentToProbability(value)
		return true
	case "volcano_eruption_chance_base":
		w.cfg.Params.VolcanoEruptionChanceBase = percentToProbability(value)
		return true
	default:
		return false
	}
}

// New returns an Ecology simulation with the provided dimensions using defaults.
func New(w, h int) *World {
	cfg := DefaultConfig()
	cfg.Width = w
	cfg.Height = h
	return NewWithConfig(cfg)
}

// NewWithConfig returns an Ecology world configured from the provided options.
func NewWithConfig(cfg Config) *World {
	total := cfg.Width * cfg.Height
	if total < 0 {
		total = 0
	}
	w := &World{
		cfg:            cfg,
		w:              cfg.Width,
		h:              cfg.Height,
		groundCurr:     make([]Ground, total),
		groundNext:     make([]Ground, total),
		vegCurr:        make([]Vegetation, total),
		vegNext:        make([]Vegetation, total),
		lavaHeight:     make([]uint8, total),
		lavaHeightNext: make([]uint8, total),
		lavaTemp:       make([]float32, total),
		lavaTempNext:   make([]float32, total),
		lavaDir:        make([]int8, total),
		lavaDirNext:    make([]int8, total),
		lavaTip:        make([]bool, total),
		lavaTipNext:    make([]bool, total),
		lavaForce:      make([]bool, total),
		lavaForceNext:  make([]bool, total),
		lavaChannel:    make([]float32, total),
		lavaElevation:  make([]int16, total),
		lavaNoise:      make([]int8, total),
		burnTTL:        make([]uint8, total),
		burnNext:       make([]uint8, total),
		rainCurr:       make([]float32, total),
		rainNext:       make([]float32, total),
		rainScratch:    make([]float32, total),
		volCurr:        make([]float32, total),
		volNext:        make([]float32, total),
		tectonic:       loadTectonicMap(cfg.Width, cfg.Height),
		display:        make([]uint8, total),
		rng:            rand.New(rand.NewSource(cfg.Seed)),
		lavaReservoir:  lavaReservoir{cells: make([]int, 0)},
	}
	return w
}

// Name returns the simulation identifier.
func (w *World) Name() string { return "ecology" }

// Size reports the grid dimensions.
func (w *World) Size() core.Size { return core.Size{W: w.w, H: w.h} }

// Cells exposes the current display buffer.
func (w *World) Cells() []uint8 { return w.display }

// Ground exposes the active ground layer.
func (w *World) Ground() []Ground { return w.groundCurr }

// Vegetation exposes the active vegetation layer.
func (w *World) Vegetation() []Vegetation { return w.vegCurr }

// RainMask exposes the current rain influence map.
func (w *World) RainMask() []float32 { return w.rainCurr }

// VolcanoMask exposes the current volcano influence map.
func (w *World) VolcanoMask() []float32 { return w.volCurr }

// WindVectorAt samples the prevailing wind vector at the provided world
// coordinate. The coordinate is expressed in cell units where integer values
// fall on tile boundaries and `.5` values represent tile centres.
func (w *World) WindVectorAt(x, y float64) (float64, float64) {
	if w == nil {
		return 0, 0
	}
	return w.windVector(x, y)
}

// TectonicMap exposes the static tectonic baseline values.
func (w *World) TectonicMap() []float32 { return w.tectonic }

// Reset prepares the initial world using deterministic randomness.
func (w *World) Reset(seed int64) {
	if w.w == 0 || w.h == 0 {
		return
	}
	effective := seed
	if effective == 0 {
		effective = w.cfg.Seed
	}
	w.rng.Seed(effective)
	w.windPhase = 0
	total := w.w * w.h
	for i := 0; i < total; i++ {
		w.groundCurr[i] = GroundDirt
		w.groundNext[i] = GroundDirt
		w.vegCurr[i] = VegetationNone
		w.vegNext[i] = VegetationNone
		w.lavaHeight[i] = 0
		w.lavaHeightNext[i] = 0
		w.lavaTemp[i] = 0
		w.lavaTempNext[i] = 0
		w.lavaDir[i] = -1
		w.lavaDirNext[i] = -1
		w.lavaTip[i] = false
		w.lavaTipNext[i] = false
		w.lavaForce[i] = false
		w.lavaForceNext[i] = false
		w.lavaChannel[i] = 0
		w.lavaElevation[i] = 0
		if i < len(w.lavaNoise) {
			w.lavaNoise[i] = int8(w.rng.Intn(3)) - 1
		}
		w.burnTTL[i] = 0
		w.burnNext[i] = 0
		w.rainCurr[i] = 0
		w.rainNext[i] = 0
		if i < len(w.rainScratch) {
			w.rainScratch[i] = 0
		}
		w.volCurr[i] = 0
		w.volNext[i] = 0
		w.display[i] = uint8(GroundDirt)
	}

	w.sprinkleRock()
	w.seedGrassPatches()
	copy(w.groundNext, w.groundCurr)
	copy(w.vegNext, w.vegCurr)

	w.metrics = VegetationMetrics{}
	w.updateMetrics(w.vegCurr)

	w.rebuildDisplay()

	w.rainRegions = w.rainRegions[:0]
	w.volcanoRegions = w.volcanoRegions[:0]
	w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]
	w.lavaVents = w.lavaVents[:0]
	w.lavaReservoir.reset()
	w.lavaTipQueue = w.lavaTipQueue[:0]
	w.lavaFailedTips = w.lavaFailedTips[:0]
	w.lavaAdvancedCells = w.lavaAdvancedCells[:0]
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func percentToProbability(value float64) float64 {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	return value / 100
}

// Step advances the simulation by applying the vegetation succession rules once.
func (w *World) Step() {
	if w.w == 0 || w.h == 0 {
		return
	}

	w.windPhase++

	w.updateRainMask()
	w.updateVolcanoMask()
	w.applyUplift()
	w.applyEruptions()
	w.applyLava()
	w.applyFire()

	grassNeighbors, shrubNeighbors := w.mooreNeighborCounts()

	thresholdGrass := uint8(w.cfg.Params.GrassNeighborThreshold)
	thresholdShrub := uint8(w.cfg.Params.ShrubNeighborThreshold)
	thresholdTree := uint8(w.cfg.Params.TreeNeighborThreshold)

	total := w.w * w.h
	for i := 0; i < total; i++ {
		current := w.vegCurr[i]
		next := current

		if w.burnTTL[i] > 0 {
			w.vegNext[i] = next
			continue
		}

		switch current {
		case VegetationNone:
			if w.groundCurr[i] == GroundDirt && grassNeighbors[i] >= thresholdGrass {
				if w.rng.Float64() < w.cfg.Params.GrassSpreadChance {
					next = VegetationGrass
				}
			}
		case VegetationGrass:
			if grassNeighbors[i] >= thresholdShrub {
				if w.rng.Float64() < w.cfg.Params.ShrubGrowthChance {
					next = VegetationShrub
				}
			}
		case VegetationShrub:
			if shrubNeighbors[i] >= thresholdTree {
				if w.rng.Float64() < w.cfg.Params.TreeGrowthChance {
					next = VegetationTree
				}
			}
		}

		w.vegNext[i] = next
	}

	w.updateMetrics(w.vegNext)
	w.vegCurr, w.vegNext = w.vegNext, w.vegCurr

	w.spawnRainRegion()
	w.spawnVolcanoProtoRegion()

	w.rebuildDisplay()
}

func (w *World) updateRainMask() {
	total := w.w * w.h
	if total == 0 || len(w.rainCurr) != total || len(w.rainNext) != total {
		return
	}

	for i := range w.rainNext {
		w.rainNext[i] = 0
	}

	regions := w.rainRegions
	active := w.rainRegions[:0]
	for i := range regions {
		region := regions[i]
		if region.ttl <= 0 {
			continue
		}
		region = w.advanceRainRegion(regions, i)
		active = append(active, region)
	}

	w.rainRegions = active

	if len(w.rainRegions) > 1 {
		merged := make([]bool, len(w.rainRegions))
		for i := 0; i < len(w.rainRegions); i++ {
			for j := i + 1; j < len(w.rainRegions); j++ {
				overlap := rainOverlapRatio(w.rainRegions[i], w.rainRegions[j])
				if overlap <= 0.15 {
					continue
				}
				larger := i
				smaller := j
				if w.rainRegions[j].area() > w.rainRegions[i].area() {
					larger, smaller = j, i
				}

				gain := clamp01(w.rainRegions[larger].targetBaseStrength + 0.1)
				w.rainRegions[larger].targetBaseStrength = gain
				if w.rainRegions[larger].mergeTicks < 8 {
					w.rainRegions[larger].mergeTicks = 8
				}
				if w.rainRegions[larger].ttl < 8 {
					w.rainRegions[larger].ttl = 8
				}
				growX := math.Max(w.rainRegions[larger].radiusX, w.rainRegions[smaller].radiusX*0.8)
				growY := math.Max(w.rainRegions[larger].radiusY, w.rainRegions[smaller].radiusY*0.8)
				if w.rainRegions[larger].targetRadiusX < growX {
					w.rainRegions[larger].targetRadiusX = growX
				}
				if w.rainRegions[larger].targetRadiusY < growY {
					w.rainRegions[larger].targetRadiusY = growY
				}

				if !merged[smaller] {
					merged[smaller] = true
					if w.rainRegions[smaller].ttl > 8 {
						w.rainRegions[smaller].ttl = 8
					}
				}
				if w.rainRegions[smaller].mergeTicks < 8 {
					w.rainRegions[smaller].mergeTicks = 8
				}
				w.rainRegions[smaller].targetBaseStrength = 0
				w.rainRegions[smaller].targetRadiusX = w.rainRegions[smaller].radiusX * 0.75
				w.rainRegions[smaller].targetRadiusY = w.rainRegions[smaller].radiusY * 0.75
			}
		}
	}

	for i := range w.rainRegions {
		w.rasterizeRainRegion(&w.rainRegions[i])
	}

	nextRegions := w.rainRegions[:0]
	for i := range w.rainRegions {
		region := w.rainRegions[i]
		region.ttl--
		if region.ttl > 0 {
			if region.targetBaseStrength == 0 && region.baseStrength < 0.02 {
				continue
			}
			nextRegions = append(nextRegions, region)
		}
	}

	w.rainRegions = nextRegions
	w.applyRainMorphology()

	w.rainCurr, w.rainNext = w.rainNext, w.rainCurr
}

func (w *World) rasterizeRainRegion(region *rainRegion) {
	if region == nil {
		return
	}
	if region.radiusX <= 0 || region.radiusY <= 0 {
		return
	}

	padding := 2.0
	minX := int(math.Floor(region.cx - region.radiusX - padding))
	maxX := int(math.Ceil(region.cx + region.radiusX + padding))
	minY := int(math.Floor(region.cy - region.radiusY - padding))
	maxY := int(math.Ceil(region.cy + region.radiusY + padding))

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= w.w {
		maxX = w.w - 1
	}
	if maxY >= w.h {
		maxY = w.h - 1
	}

	if minX > maxX || minY > maxY {
		return
	}

	cosA := math.Cos(region.angle)
	sinA := math.Sin(region.angle)
	invRadiusX := 1.0 / region.radiusX
	invRadiusY := 1.0 / region.radiusY
	falloff := region.falloff
	if falloff <= 0 {
		falloff = 1.15
	}

	strength := clamp01(region.strength)
	threshold := clamp01(region.threshold)
	noiseScale := region.noiseScale
	if noiseScale <= 0 {
		noiseScale = 0.08
	}
	stretchX := region.noiseStretchX
	if stretchX == 0 {
		stretchX = 1
	}
	stretchY := region.noiseStretchY
	if stretchY == 0 {
		stretchY = 1
	}

	for y := minY; y <= maxY; y++ {
		cy := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			cx := float64(x) + 0.5
			dx := cx - region.cx
			dy := cy - region.cy

			rx := dx*cosA + dy*sinA
			ry := -dx*sinA + dy*cosA

			nx := (rx * stretchX) * noiseScale
			ny := (ry * stretchY) * noiseScale
			nx += region.noiseOffsetX
			ny += region.noiseOffsetY
			n := fbmNoise2D(nx, ny, 3, 0.5, 1.9, region.noiseSeed)

			distX := rx * invRadiusX
			distY := ry * invRadiusY
			radial := math.Sqrt(distX*distX + distY*distY)
			if radial > 1 {
				continue
			}

			// SOFT CLOUD MASK
			c := smoothstep(threshold-0.1, threshold+0.1, n)
			if c <= 0 {
				continue
			}

			// RADIAL CORE FLOOR (prevents central holes)
			if radial < 0.45 {
				c = 1
			}

			// final value uses soft mask instead of step
			fall := smoothstep(0, 1, 1-math.Pow(radial, falloff))
			val := float32(clamp01(float64(c) * fall * strength))
			if val <= 0.01 {
				continue
			}

			idx := y*w.w + x
			if idx < 0 || idx >= len(w.rainNext) {
				continue
			}
			if val <= w.rainNext[idx] {
				continue
			}
			w.rainNext[idx] = val
		}
	}
}

func (w *World) spawnRainRegion() {
	if w.w <= 0 || w.h <= 0 {
		return
	}

	maxRegions := w.cfg.Params.RainMaxRegions
	if maxRegions <= 0 {
		return
	}

	available := maxRegions - len(w.rainRegions)
	if available <= 0 {
		return
	}

	spawnChance := clampFloat(w.cfg.Params.RainSpawnChance, 0, 1)
	if spawnChance <= 0 {
		return
	}

	attempts := available
	if attempts > 2 {
		attempts = 2
	}

	total := w.w * w.h
	if total <= 0 {
		return
	}

	coverage := w.currentRainCoverageRatio()
	hasActive := len(w.rainRegions) > 0
	for i := 0; i < attempts; i++ {
		if hasActive && coverage > 0.15 {
			skipChance := clampFloat((coverage-0.15)*3, 0, 0.9)
			if w.rng.Float64() < skipChance {
				continue
			}
		}
		if w.rng.Float64() >= spawnChance {
			continue
		}
		region := w.makeRainRegion()
		w.rainRegions = append(w.rainRegions, region)
		coverage += region.area() / float64(total)
		hasActive = len(w.rainRegions) > 0
		if len(w.rainRegions) >= maxRegions {
			break
		}
	}
}

func (w *World) advanceRainRegion(regions []rainRegion, index int) rainRegion {
	if index < 0 || index >= len(regions) {
		return rainRegion{}
	}

	region := regions[index]
	region.age++

	if region.targetBaseStrength == 0 && region.baseStrength > 0 {
		region.targetBaseStrength = region.baseStrength
	}
	if region.targetRadiusX == 0 {
		region.targetRadiusX = region.radiusX
	}
	if region.targetRadiusY == 0 {
		region.targetRadiusY = region.radiusY
	}

	blend := 0.125
	if region.mergeTicks > 0 {
		blend = 0.25
		region.mergeTicks--
	}

	region.baseStrength = lerp(region.baseStrength, region.targetBaseStrength, blend)
	region.radiusX = lerp(region.radiusX, region.targetRadiusX, blend)
	region.radiusY = lerp(region.radiusY, region.targetRadiusY, blend)
	if region.radiusX < 1 {
		region.radiusX = 1
	}
	if region.radiusY < 1 {
		region.radiusY = 1
	}

	targetVX, targetVY := w.windVector(region.cx, region.cy)
	const inertia = 0.08
	region.vx = lerp(region.vx, targetVX, inertia)
	region.vy = lerp(region.vy, targetVY, inertia)

	const cohesionRadius = 50.0
	const cohesionBlend = 0.08
	var sumVX, sumVY float64
	var count int
	for i := range regions {
		if i == index {
			continue
		}
		neighbor := regions[i]
		if neighbor.ttl <= 0 {
			continue
		}
		if math.Hypot(neighbor.cx-region.cx, neighbor.cy-region.cy) > cohesionRadius {
			continue
		}
		sumVX += neighbor.vx
		sumVY += neighbor.vy
		count++
	}
	if count > 0 {
		avgVX := sumVX / float64(count)
		avgVY := sumVY / float64(count)
		region.vx = lerp(region.vx, avgVX, cohesionBlend)
		region.vy = lerp(region.vy, avgVY, cohesionBlend)
	}

	const maxStep = 0.8
	speed := math.Hypot(region.vx, region.vy)
	if speed > maxStep {
		scale := maxStep / speed
		region.vx *= scale
		region.vy *= scale
		speed = maxStep
	}

	dx := region.vx
	dy := region.vy

	region.noiseOffsetX += dx * region.noiseScale
	region.noiseOffsetY += dy * region.noiseScale

	region.cx += dx
	region.cy += dy

	marginX := region.radiusX
	if marginX < 1 {
		marginX = 1
	}
	marginY := region.radiusY
	if marginY < 1 {
		marginY = 1
	}
	if region.cx < -marginX {
		region.cx = -marginX
	} else if region.cx > float64(w.w)+marginX {
		region.cx = float64(w.w) + marginX
	}
	if region.cy < -marginY {
		region.cy = -marginY
	} else if region.cy > float64(w.h)+marginY {
		region.cy = float64(w.h) + marginY
	}

	if region.maxTTL < region.ttl {
		region.maxTTL = region.ttl
	}
	if region.maxTTL <= 0 {
		region.maxTTL = 1
	}

	progress := float64(region.age) / float64(region.maxTTL)
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	envelope := math.Sin(progress * math.Pi)
	if envelope < 0 {
		envelope = 0
	}

	swing := clampFloat(region.strengthVariation, 0, 0.5)
	factor := 1 - swing + envelope*2*swing
	targetStrength := clamp01(region.baseStrength * factor)
	region.strength = lerp(region.strength, targetStrength, 0.1)

	if region.preset != rainPresetPuffy {
		if speed > 1e-3 {
			region.angle = math.Atan2(region.vy, region.vx)
		}
	}

	return region
}

func (w *World) applyRainMorphology() {
	total := w.w * w.h
	if total == 0 || len(w.rainNext) != total || len(w.rainScratch) != total {
		return
	}

	const closingRadius = 3
	const openingRadius = 1

	w.dilateRain(w.rainNext, w.rainScratch, closingRadius)
	w.erodeRain(w.rainScratch, w.rainNext, closingRadius, 0.02)

	w.erodeRain(w.rainNext, w.rainScratch, openingRadius, 0)
	w.dilateRain(w.rainScratch, w.rainNext, openingRadius)

	w.removeTinyRainIslands(25, 0.05)
}

func (w *World) dilateRain(src, dst []float32, radius int) {
	if radius <= 0 {
		copy(dst, src)
		return
	}
	for y := 0; y < w.h; y++ {
		for x := 0; x < w.w; x++ {
			maxVal := float32(0)
			for oy := -radius; oy <= radius; oy++ {
				ny := y + oy
				if ny < 0 || ny >= w.h {
					continue
				}
				for ox := -radius; ox <= radius; ox++ {
					nx := x + ox
					if nx < 0 || nx >= w.w {
						continue
					}
					v := src[ny*w.w+nx]
					if v > maxVal {
						maxVal = v
					}
				}
			}
			dst[y*w.w+x] = maxVal
		}
	}
}

func (w *World) erodeRain(src, dst []float32, radius int, floor float32) {
	if radius <= 0 {
		copy(dst, src)
		return
	}
	for y := 0; y < w.h; y++ {
		for x := 0; x < w.w; x++ {
			minVal := float32(1)
			hasVal := false
			for oy := -radius; oy <= radius; oy++ {
				ny := y + oy
				if ny < 0 || ny >= w.h {
					continue
				}
				for ox := -radius; ox <= radius; ox++ {
					nx := x + ox
					if nx < 0 || nx >= w.w {
						continue
					}
					v := src[ny*w.w+nx]
					if !hasVal || v < minVal {
						minVal = v
						hasVal = true
					}
				}
			}
			if !hasVal {
				minVal = 0
			}
			if minVal < floor {
				minVal = 0
			}
			dst[y*w.w+x] = minVal
		}
	}
}

func (w *World) removeTinyRainIslands(minArea int, threshold float32) {
	total := w.w * w.h
	if total == 0 || len(w.rainNext) != total {
		return
	}
	visited := make([]bool, total)
	component := make([]int, 0, minArea)
	queue := make([]int, 0, minArea)
	neighbors := [8][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}}

	for idx := 0; idx < total; idx++ {
		if visited[idx] || w.rainNext[idx] <= threshold {
			continue
		}
		visited[idx] = true
		component = component[:0]
		component = append(component, idx)
		queue = queue[:0]
		queue = append(queue, idx)
		area := 0
		for len(queue) > 0 {
			cur := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			area++
			cx := cur % w.w
			cy := cur / w.w
			for _, n := range neighbors {
				nx := cx + n[0]
				ny := cy + n[1]
				if nx < 0 || nx >= w.w || ny < 0 || ny >= w.h {
					continue
				}
				nIdx := ny*w.w + nx
				if visited[nIdx] {
					continue
				}
				if w.rainNext[nIdx] <= threshold {
					continue
				}
				visited[nIdx] = true
				queue = append(queue, nIdx)
				component = append(component, nIdx)
			}
		}
		if area < minArea {
			for _, cIdx := range component {
				w.rainNext[cIdx] = 0
			}
		}
	}
}

func (w *World) currentRainCoverageRatio() float64 {
	total := w.w * w.h
	if total == 0 || len(w.rainCurr) != total {
		return 0
	}
	covered := 0
	for _, v := range w.rainCurr {
		if v > 0.05 {
			covered++
		}
	}
	if covered == 0 {
		return 0
	}
	return float64(covered) / float64(total)
}

func (w *World) makeRainRegion() rainRegion {
	params := w.cfg.Params

	radiusMin := params.RainRadiusMin
	if radiusMin < 16 {
		radiusMin = 16
	}
	radiusMax := params.RainRadiusMax
	if radiusMax < radiusMin {
		radiusMax = radiusMin
	}

	baseRadius := float64(radiusMin)
	if radiusMax > radiusMin {
		baseRadius = float64(radiusMin + w.rng.Intn(radiusMax-radiusMin+1))
	}

	ttlMin := params.RainTTLMin
	if ttlMin < 1 {
		ttlMin = 1
	}
	ttlMax := params.RainTTLMax
	if ttlMax < ttlMin {
		ttlMax = ttlMin
	}

	strengthMin := clampFloat(params.RainStrengthMin, 0, 1)
	strengthMax := clampFloat(params.RainStrengthMax, 0, 1)
	if strengthMax < strengthMin {
		strengthMax = strengthMin
	}
	if strengthMin < 0.5 {
		strengthMin = 0.5
	}

	baseStrength := strengthMin
	if strengthMax > strengthMin {
		baseStrength += w.rng.Float64() * (strengthMax - strengthMin)
	}

	threshold := 0.42 + w.rng.Float64()*0.1
	falloff := 1.15 + w.rng.Float64()*0.05
	noiseScale := 0.075 + w.rng.Float64()*0.01
	stretchX := 1.0
	stretchY := 1.0
	radiusX := baseRadius
	radiusY := baseRadius
	ttl := ttlMin
	if ttlMax > ttlMin {
		ttl += w.rng.Intn(ttlMax - ttlMin + 1)
	}

	presetRoll := w.rng.Float64()
	preset := rainPresetPuffy
	switch {
	case presetRoll < 0.55:
		preset = rainPresetPuffy
		falloff = 1.12 + w.rng.Float64()*0.08
	case presetRoll < 0.85:
		preset = rainPresetStratus
		radiusX = baseRadius * (1.1 + w.rng.Float64()*0.4)
		radiusY = baseRadius * 0.6
		stretchY = 0.6
		noiseScale = 0.06 + w.rng.Float64()*0.015
		falloff = 1.08 + w.rng.Float64()*0.08
	default:
		preset = rainPresetSquall
		radiusX = 40 + w.rng.Float64()*20
		radiusY = 10 + w.rng.Float64()*6
		if radiusX > float64(params.RainRadiusMax)*1.5 {
			radiusX = float64(params.RainRadiusMax) * 1.5
		}
		stretchX = 1.2
		stretchY = 0.8
		noiseScale = 0.065 + w.rng.Float64()*0.02
		falloff = 1.05 + w.rng.Float64()*0.08
		ttl = 8 + w.rng.Intn(8)
	}

	if radiusX < 10 {
		radiusX = 10
	}
	if radiusY < 10 {
		radiusY = 10
	}

	maxSpanX := math.Max(1, float64(w.w))
	maxSpanY := math.Max(1, float64(w.h))
	if radiusX > maxSpanX {
		radiusX = maxSpanX
	}
	if radiusY > maxSpanY {
		radiusY = maxSpanY
	}

	cx := float64(w.rng.Intn(w.w)) + 0.5
	cy := float64(w.rng.Intn(w.h)) + 0.5
	seed := w.rng.Int63()
	vx, vy := w.windVector(cx, cy)

	strengthVariation := 0.1 + w.rng.Float64()*0.1
	noiseOffsetX := (w.rng.Float64()*2 - 1) * 5
	noiseOffsetY := (w.rng.Float64()*2 - 1) * 5

	region := rainRegion{
		cx:                 cx,
		cy:                 cy,
		radiusX:            radiusX,
		radiusY:            radiusY,
		baseStrength:       baseStrength,
		strength:           baseStrength,
		strengthVariation:  strengthVariation,
		targetBaseStrength: baseStrength,
		targetRadiusX:      radiusX,
		targetRadiusY:      radiusY,
		ttl:                ttl,
		maxTTL:             ttl,
		vx:                 vx,
		vy:                 vy,
		threshold:          threshold,
		falloff:            falloff,
		noiseScale:         noiseScale,
		noiseStretchX:      stretchX,
		noiseStretchY:      stretchY,
		noiseSeed:          seed,
		preset:             preset,
		noiseOffsetX:       noiseOffsetX,
		noiseOffsetY:       noiseOffsetY,
	}

	if preset != rainPresetPuffy {
		speed := math.Hypot(vx, vy)
		if speed > 1e-3 {
			region.angle = math.Atan2(vy, vx)
		}
	}

	return region
}

const (
	windCurlEpsilon = 1e-6
)

func (w *World) windVector(x, y float64) (float64, float64) {
	scale := w.cfg.Params.WindNoiseScale
	speed := w.cfg.Params.WindSpeedScale
	if scale <= 0 || speed <= 0 {
		return 0, 0
	}

	phase := w.windPhase * w.cfg.Params.WindTemporalScale
	h := 1.0 / scale

	phiXP := w.windPotentialAt(x+h, y, scale, phase)
	phiXM := w.windPotentialAt(x-h, y, scale, phase)
	phiYP := w.windPotentialAt(x, y+h, scale, phase)
	phiYM := w.windPotentialAt(x, y-h, scale, phase)

	dphidx := (phiXP - phiXM) / (2 * h)
	dphidy := (phiYP - phiYM) / (2 * h)

	curlX := dphidy
	curlY := -dphidx
	magnitude := math.Hypot(curlX, curlY)
	if magnitude < windCurlEpsilon {
		return 0, 0
	}

	invMag := 1.0 / magnitude
	vx := curlX * invMag * speed
	vy := curlY * invMag * speed

	return vx, vy
}

func (w *World) windPotentialAt(x, y, scale, phase float64) float64 {
	return fbmNoise3D(x*scale, y*scale, phase, 4, 0.5, 1.9, w.cfg.Seed)
}

func fbmNoise3D(x, y, z float64, octaves int, gain, lacunarity float64, seed int64) float64 {
	if octaves <= 0 {
		return 0.5
	}
	amplitude := 1.0
	frequency := 1.0
	sum := 0.0
	ampAccum := 0.0
	for i := 0; i < octaves; i++ {
		n := perlin3D(x*frequency, y*frequency, z*frequency, seed+int64(i)*67)
		sum += n * amplitude
		ampAccum += amplitude
		amplitude *= gain
		frequency *= lacunarity
	}
	if ampAccum == 0 {
		return 0.5
	}
	value := sum / ampAccum
	return value*0.5 + 0.5
}

func fbmNoise2D(x, y float64, octaves int, gain, lacunarity float64, seed int64) float64 {
	if octaves <= 0 {
		return 0.5
	}
	amplitude := 1.0
	frequency := 1.0
	sum := 0.0
	ampAccum := 0.0
	for i := 0; i < octaves; i++ {
		n := perlin2D(x*frequency, y*frequency, seed+int64(i)*57)
		sum += n * amplitude
		ampAccum += amplitude
		amplitude *= gain
		frequency *= lacunarity
	}
	if ampAccum == 0 {
		return 0.5
	}
	value := sum / ampAccum
	return value*0.5 + 0.5
}

var perlin3DGradients = [...]struct{ x, y, z float64 }{
	{1, 1, 0},
	{-1, 1, 0},
	{1, -1, 0},
	{-1, -1, 0},
	{1, 0, 1},
	{-1, 0, 1},
	{1, 0, -1},
	{-1, 0, -1},
	{0, 1, 1},
	{0, -1, 1},
	{0, 1, -1},
	{0, -1, -1},
}

func perlin3D(x, y, z float64, seed int64) float64 {
	x0 := math.Floor(x)
	y0 := math.Floor(y)
	z0 := math.Floor(z)

	xf := x - x0
	yf := y - y0
	zf := z - z0

	ix0 := int64(x0)
	iy0 := int64(y0)
	iz0 := int64(z0)
	ix1 := ix0 + 1
	iy1 := iy0 + 1
	iz1 := iz0 + 1

	g000 := gradDot3D(ix0, iy0, iz0, xf, yf, zf, seed)
	g100 := gradDot3D(ix1, iy0, iz0, xf-1, yf, zf, seed)
	g010 := gradDot3D(ix0, iy1, iz0, xf, yf-1, zf, seed)
	g110 := gradDot3D(ix1, iy1, iz0, xf-1, yf-1, zf, seed)
	g001 := gradDot3D(ix0, iy0, iz1, xf, yf, zf-1, seed)
	g101 := gradDot3D(ix1, iy0, iz1, xf-1, yf, zf-1, seed)
	g011 := gradDot3D(ix0, iy1, iz1, xf, yf-1, zf-1, seed)
	g111 := gradDot3D(ix1, iy1, iz1, xf-1, yf-1, zf-1, seed)

	u := fade(xf)
	v := fade(yf)
	w := fade(zf)

	x00 := lerp(g000, g100, u)
	x10 := lerp(g010, g110, u)
	x01 := lerp(g001, g101, u)
	x11 := lerp(g011, g111, u)

	interpY0 := lerp(x00, x10, v)
	interpY1 := lerp(x01, x11, v)

	return lerp(interpY0, interpY1, w)
}

func gradDot3D(ix, iy, iz int64, dx, dy, dz float64, seed int64) float64 {
	grad := perlin3DGradients[hash3D(ix, iy, iz, seed)%uint32(len(perlin3DGradients))]
	return grad.x*dx + grad.y*dy + grad.z*dz
}

func hash3D(x, y, z, seed int64) uint32 {
	n := uint64(x)*0x9e3779b97f4a7c15 + uint64(y)*0xbf58476d1ce4e5b9 + uint64(z)*0x94d049bb133111eb + uint64(seed)*0xda942042e4dd58b5
	n = (n ^ (n >> 30)) * 0xbf58476d1ce4e5b9
	n = (n ^ (n >> 27)) * 0x94d049bb133111eb
	n ^= n >> 31
	return uint32(n & 0xffffffff)
}

func perlin2D(x, y float64, seed int64) float64 {
	x0 := math.Floor(x)
	y0 := math.Floor(y)
	xf := x - x0
	yf := y - y0

	ix0 := int64(x0)
	iy0 := int64(y0)
	ix1 := ix0 + 1
	iy1 := iy0 + 1

	g00 := gradDot(ix0, iy0, xf, yf, seed)
	g10 := gradDot(ix1, iy0, xf-1, yf, seed)
	g01 := gradDot(ix0, iy1, xf, yf-1, seed)
	g11 := gradDot(ix1, iy1, xf-1, yf-1, seed)

	u := fade(xf)
	v := fade(yf)

	return lerp(lerp(g00, g10, u), lerp(g01, g11, u), v)
}

func gradDot(ix, iy int64, dx, dy float64, seed int64) float64 {
	h := hash2D(ix, iy, seed)
	switch h & 7 {
	case 0:
		return dx
	case 1:
		return -dx
	case 2:
		return dy
	case 3:
		return -dy
	case 4:
		return dx + dy
	case 5:
		return dx - dy
	case 6:
		return -dx + dy
	default:
		return -dx - dy
	}
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func hash2D(x, y, seed int64) uint32 {
	n := uint64(x)*0x9e3779b97f4a7c15 + uint64(y)*0xbf58476d1ce4e5b9 + uint64(seed)*0x94d049bb133111eb
	n = (n ^ (n >> 30)) * 0xbf58476d1ce4e5b9
	n = (n ^ (n >> 27)) * 0x94d049bb133111eb
	n ^= n >> 31
	return uint32(n & 0xffffffff)
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		return 0
	}
	t := clampFloat((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func rainOverlapRatio(a, b rainRegion) float64 {
	ra := math.Sqrt(a.radiusX * a.radiusY)
	rb := math.Sqrt(b.radiusX * b.radiusY)
	if ra <= 0 || rb <= 0 {
		return 0
	}
	d := math.Hypot(a.cx-b.cx, a.cy-b.cy)
	areaA := math.Pi * ra * ra
	areaB := math.Pi * rb * rb
	intersection := circleIntersectionArea(ra, rb, d)
	if intersection <= 0 {
		return 0
	}
	smaller := areaA
	if areaB < smaller {
		smaller = areaB
	}
	if smaller == 0 {
		return 0
	}
	return intersection / smaller
}

func circleIntersectionArea(ra, rb, d float64) float64 {
	if d >= ra+rb {
		return 0
	}
	if d <= math.Abs(ra-rb) {
		smaller := math.Min(ra, rb)
		return math.Pi * smaller * smaller
	}
	ra2 := ra * ra
	rb2 := rb * rb
	part1 := ra2 * math.Acos((d*d+ra2-rb2)/(2*d*ra))
	part2 := rb2 * math.Acos((d*d+rb2-ra2)/(2*d*rb))
	part3 := 0.5 * math.Sqrt((-d+ra+rb)*(d+ra-rb)*(d-ra+rb)*(d+ra+rb))
	return part1 + part2 - part3
}

func (r rainRegion) area() float64 {
	if r.radiusX <= 0 || r.radiusY <= 0 {
		return 0
	}
	return math.Pi * r.radiusX * r.radiusY
}

func (w *World) updateVolcanoMask() {
	total := w.w * w.h
	if total == 0 || len(w.volCurr) != total || len(w.volNext) != total {
		return
	}

	for i := range w.volNext {
		w.volNext[i] = 0
	}

	w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]

	nextRegions := w.volcanoRegions[:0]
	for i := range w.volcanoRegions {
		region := w.volcanoRegions[i]
		if region.ttl <= 0 {
			w.expiredVolcanoProtos = append(w.expiredVolcanoProtos, region)
			continue
		}
		w.rasterizeVolcanoRegion(region)
		region.ttl--
		if region.ttl > 0 {
			nextRegions = append(nextRegions, region)
		} else {
			w.expiredVolcanoProtos = append(w.expiredVolcanoProtos, region)
		}
	}

	w.volcanoRegions = nextRegions
	w.volCurr, w.volNext = w.volNext, w.volCurr
}

func (w *World) rasterizeVolcanoRegion(region volcanoProtoRegion) {
	if region.radius <= 0 {
		return
	}

	minX := int(math.Floor(region.cx - region.radius))
	maxX := int(math.Ceil(region.cx + region.radius))
	minY := int(math.Floor(region.cy - region.radius))
	maxY := int(math.Ceil(region.cy + region.radius))

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= w.w {
		maxX = w.w - 1
	}
	if maxY >= w.h {
		maxY = w.h - 1
	}

	radius := region.radius
	invRadius := 1.0 / radius

	for y := minY; y <= maxY; y++ {
		cy := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			cx := float64(x) + 0.5
			dx := cx - region.cx
			dy := cy - region.cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > radius {
				continue
			}
			value := region.strength * (1 - dist*invRadius)
			if value <= 0 {
				continue
			}
			if value > 1 {
				value = 1
			}
			idx := y*w.w + x
			if idx < 0 || idx >= len(w.volNext) {
				continue
			}
			if current := w.volNext[idx]; current >= float32(value) {
				continue
			}
			w.volNext[idx] = float32(value)
		}
	}
}

func (w *World) applyUplift() {
	total := w.w * w.h
	if total == 0 || len(w.groundCurr) != total || len(w.groundNext) != total {
		return
	}

	for i := 0; i < total; i++ {
		w.groundNext[i] = w.groundCurr[i]
	}

	baseChance := w.cfg.Params.VolcanoUpliftChanceBase
	if baseChance <= 0 {
		w.groundCurr, w.groundNext = w.groundNext, w.groundCurr
		return
	}

	for i := 0; i < total; i++ {
		if w.groundCurr[i] != GroundRock {
			continue
		}
		if i >= len(w.volCurr) {
			continue
		}
		mask := float64(w.volCurr[i])
		if mask <= 0 {
			continue
		}
		chance := baseChance * mask
		if chance > 1 {
			chance = 1
		}
		if chance <= 0 {
			continue
		}
		if w.rng.Float64() >= chance {
			continue
		}
		w.groundNext[i] = GroundMountain
		if i < len(w.display) {
			w.display[i] = uint8(GroundMountain)
		}
	}

	w.groundCurr, w.groundNext = w.groundNext, w.groundCurr
}

func (w *World) applyEruptions() {
	if len(w.expiredVolcanoProtos) == 0 {
		return
	}

	total := w.w * w.h
	if total == 0 || len(w.groundCurr) != total {
		w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]
		return
	}
	if len(w.lavaHeight) != total || len(w.lavaTemp) != total || len(w.lavaElevation) != total {
		w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]
		return
	}
	if len(w.volCurr) != total {
		w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]
		return
	}

	baseChance := w.cfg.Params.VolcanoEruptionChanceBase
	if baseChance <= 0 {
		w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]
		return
	}

	for _, region := range w.expiredVolcanoProtos {
		if region.radius <= 0 {
			continue
		}
		maskMean := w.volcanoMaskMean(region)
		if maskMean <= 0 {
			continue
		}
		chance := baseChance * maskMean
		if chance > 1 {
			chance = 1
		}
		if w.rng.Float64() >= chance {
			continue
		}

		w.eruptRegion(region)
	}

	w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]
}

func (w *World) volcanoMaskMean(region volcanoProtoRegion) float64 {
	if region.radius <= 0 {
		return 0
	}
	total := w.w * w.h
	if total == 0 || len(w.volCurr) != total {
		return 0
	}

	minX := int(math.Floor(region.cx - region.radius))
	maxX := int(math.Ceil(region.cx + region.radius))
	minY := int(math.Floor(region.cy - region.radius))
	maxY := int(math.Ceil(region.cy + region.radius))

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= w.w {
		maxX = w.w - 1
	}
	if maxY >= w.h {
		maxY = w.h - 1
	}

	var sum float64
	var count int
	radius := region.radius

	for y := minY; y <= maxY; y++ {
		cy := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			cx := float64(x) + 0.5
			dx := cx - region.cx
			dy := cy - region.cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > radius {
				continue
			}
			idx := y*w.w + x
			if idx < 0 || idx >= len(w.volCurr) {
				continue
			}
			sum += float64(w.volCurr[idx])
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (w *World) eruptRegion(region volcanoProtoRegion) {
	if region.radius <= 0 {
		return
	}

	w.lavaReservoir.reset()
	w.clearLavaField()
	w.lavaVents = w.lavaVents[:0]
	for i := range w.lavaChannel {
		w.lavaChannel[i] = 0
	}
	w.buildLavaElevation(region)

	minX := int(math.Floor(region.cx - region.radius))
	maxX := int(math.Ceil(region.cx + region.radius))
	minY := int(math.Floor(region.cy - region.radius))
	maxY := int(math.Ceil(region.cy + region.radius))

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= w.w {
		maxX = w.w - 1
	}
	if maxY >= w.h {
		maxY = w.h - 1
	}

	coreRadius := region.radius * 0.35
	rimRadius := region.radius * 0.9
	if rimRadius < coreRadius {
		rimRadius = coreRadius
	}

	coreCells := w.lavaTipQueue[:0]

	for y := minY; y <= maxY; y++ {
		cy := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			cx := float64(x) + 0.5
			dx := cx - region.cx
			dy := cy - region.cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > region.radius {
				continue
			}
			idx := y*w.w + x
			if idx < 0 || idx >= len(w.groundCurr) {
				continue
			}

			switch {
			case dist <= coreRadius:
				height := 2 + w.rng.Intn(2)
				w.setLavaCell(idx, height, 1, -1, false)
				coreCells = append(coreCells, idx)
			case dist <= rimRadius:
				if w.groundCurr[idx] == GroundRock {
					w.groundCurr[idx] = GroundMountain
					if idx < len(w.display) {
						w.display[idx] = uint8(GroundMountain)
					}
				}
			}
		}
	}

	if len(coreCells) == 0 {
		cx := int(math.Round(region.cx))
		cy := int(math.Round(region.cy))
		if cx < 0 {
			cx = 0
		}
		if cy < 0 {
			cy = 0
		}
		if cx >= w.w {
			cx = w.w - 1
		}
		if cy >= w.h {
			cy = w.h - 1
		}
		if cx >= 0 && cy >= 0 {
			centerIdx := cy*w.w + cx
			if centerIdx >= 0 && centerIdx < len(w.groundCurr) {
				coreCells = append(coreCells, centerIdx)
			}
		}
	}

	vents := 1 + w.rng.Intn(3)
	if vents > len(coreCells) {
		vents = len(coreCells)
	}
	if vents <= 0 {
		return
	}

	w.rng.Shuffle(len(coreCells), func(i, j int) {
		coreCells[i], coreCells[j] = coreCells[j], coreCells[i]
	})

	w.lavaVents = w.lavaVents[:0]
	lifeMin := w.cfg.Params.LavaLifeMin
	if lifeMin < 1 {
		lifeMin = 1
	}
	lifeMax := w.cfg.Params.LavaLifeMax
	if lifeMax < lifeMin {
		lifeMax = lifeMin
	}

	ttlSum := 0
	for i := 0; i < vents; i++ {
		idx := coreCells[i]
		ttl := lifeMin
		if lifeMax > lifeMin {
			ttl += w.rng.Intn(lifeMax - lifeMin + 1)
		}
		ttlSum += ttl
		outIdx, dir, downhill := w.pickDownhill(idx)
		if !downhill {
			if outIdx < 0 {
				outIdx = idx
			}
		}
		currentHeight := int(w.lavaHeight[idx])
		if currentHeight < 2 {
			currentHeight = 2
		}
		w.setLavaCell(idx, currentHeight, 1, dir, false)
		tip := false
		if outIdx >= 0 && outIdx < len(w.groundCurr) && outIdx != idx {
			existing := int(w.lavaHeight[outIdx])
			if existing < 1 {
				existing = 1
			}
			w.setLavaCell(outIdx, existing, 1, dir, true)
			tip = true
		}
		w.lavaVents = append(w.lavaVents, lavaVent{idx: idx, ttl: ttl, dir: dir, outIdx: outIdx})
		if tip && outIdx >= 0 && outIdx < len(w.lavaTip) {
			w.lavaTip[outIdx] = true
			if outIdx < len(w.lavaTipNext) {
				w.lavaTipNext[outIdx] = true
			}
		}
	}

	if ttlSum > 0 && len(w.lavaVents) > 0 {
		avg := ttlSum / len(w.lavaVents)
		if avg < 1 {
			avg = 1
		}
		w.lavaReservoir.assign(coreCells, avg)
	}
}

func (w *World) spawnVolcanoProtoRegion() {
	params := w.cfg.Params
	maxRegions := params.VolcanoProtoMaxRegions
	if maxRegions <= 0 {
		return
	}
	if len(w.volcanoRegions) >= maxRegions {
		return
	}
	if params.VolcanoProtoSpawnChance <= 0 {
		return
	}
	if w.rng.Float64() >= params.VolcanoProtoSpawnChance {
		return
	}

	total := w.w * w.h
	if total == 0 || len(w.tectonic) != total {
		return
	}

	attempts := 8
	bestIdx := -1
	bestScore := -1.0
	threshold := params.VolcanoProtoTectonicThreshold
	if threshold < 0 {
		threshold = 0
	}

	for i := 0; i < attempts; i++ {
		idx := w.rng.Intn(total)
		score := float64(w.tectonic[idx]) + w.rng.Float64()*0.05
		if score > bestScore {
			bestScore = score
			bestIdx = idx
		}
	}

	if bestIdx < 0 {
		return
	}

	baseValue := float64(w.tectonic[bestIdx])
	if baseValue < threshold {
		return
	}

	radiusMin := params.VolcanoProtoRadiusMin
	radiusMax := params.VolcanoProtoRadiusMax
	if radiusMin <= 0 {
		radiusMin = 1
	}
	if radiusMax < radiusMin {
		radiusMax = radiusMin
	}
	radius := radiusMin
	if radiusMax > radiusMin {
		radius += w.rng.Intn(radiusMax - radiusMin + 1)
	}

	ttlMin := params.VolcanoProtoTTLMin
	ttlMax := params.VolcanoProtoTTLMax
	if ttlMin <= 0 {
		ttlMin = 1
	}
	if ttlMax < ttlMin {
		ttlMax = ttlMin
	}
	ttl := ttlMin
	if ttlMax > ttlMin {
		ttl += w.rng.Intn(ttlMax - ttlMin + 1)
	}

	strengthMin := params.VolcanoProtoStrengthMin
	strengthMax := params.VolcanoProtoStrengthMax
	if strengthMin < 0 {
		strengthMin = 0
	}
	if strengthMax < strengthMin {
		strengthMax = strengthMin
	}
	strength := strengthMin
	if strengthMax > strengthMin {
		strength += w.rng.Float64() * (strengthMax - strengthMin)
	}
	if strength > 1 {
		strength = 1
	}

	jitter := func() float64 {
		return w.rng.Float64() - 0.5
	}

	cx := float64(bestIdx%w.w) + 0.5 + jitter()
	cy := float64(bestIdx/w.w) + 0.5 + jitter()
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	if cx > float64(w.w) {
		cx = float64(w.w)
	}
	if cy > float64(w.h) {
		cy = float64(w.h)
	}

	w.volcanoRegions = append(w.volcanoRegions, volcanoProtoRegion{
		cx:       cx,
		cy:       cy,
		radius:   float64(radius),
		strength: strength,
		ttl:      ttl,
		noise:    w.rng.Int63(),
	})
}

func (w *World) applyLava() {
	total := w.w * w.h
	if total == 0 {
		return
	}
	if len(w.groundCurr) != total || len(w.groundNext) != total {
		return
	}
	if len(w.lavaHeight) != total || len(w.lavaHeightNext) != total {
		return
	}
	if len(w.lavaTemp) != total || len(w.lavaTempNext) != total {
		return
	}
	if len(w.lavaDir) != total || len(w.lavaDirNext) != total {
		return
	}
	if len(w.lavaTip) != total || len(w.lavaTipNext) != total {
		return
	}
	if len(w.lavaForce) != total || len(w.lavaForceNext) != total {
		return
	}
	if len(w.lavaElevation) != total || len(w.lavaChannel) != total {
		return
	}

	copy(w.groundNext, w.groundCurr)
	copy(w.lavaHeightNext, w.lavaHeight)
	copy(w.lavaTempNext, w.lavaTemp)
	copy(w.lavaDirNext, w.lavaDir)
	for i := 0; i < total; i++ {
		w.lavaForceNext[i] = w.lavaForce[i]
		w.lavaTipNext[i] = false
	}

	w.lavaAdvancedCells = w.lavaAdvancedCells[:0]
	w.processLavaVents()
	w.feedCalderaReservoir()

	w.lavaTipQueue = w.lavaTipQueue[:0]
	for idx := 0; idx < total; idx++ {
		if w.lavaTip[idx] && w.groundCurr[idx] == GroundLava {
			w.lavaTipQueue = append(w.lavaTipQueue, idx)
		}
	}
	if len(w.lavaTipQueue) > 1 {
		w.rng.Shuffle(len(w.lavaTipQueue), func(i, j int) {
			w.lavaTipQueue[i], w.lavaTipQueue[j] = w.lavaTipQueue[j], w.lavaTipQueue[i]
		})
	}

	w.lavaFailedTips = w.lavaFailedTips[:0]
	for _, idx := range w.lavaTipQueue {
		if !w.advanceLavaTip(idx) {
			w.lavaFailedTips = append(w.lavaFailedTips, idx)
		}
	}

	w.poolFailedTips()
	w.coolLavaCells()
	w.reinforceLavaChannels()
	w.detectLavaTips()

	w.groundCurr, w.groundNext = w.groundNext, w.groundCurr
	w.lavaHeight, w.lavaHeightNext = w.lavaHeightNext, w.lavaHeight
	w.lavaTemp, w.lavaTempNext = w.lavaTempNext, w.lavaTemp
	w.lavaDir, w.lavaDirNext = w.lavaDirNext, w.lavaDir
	w.lavaTip, w.lavaTipNext = w.lavaTipNext, w.lavaTip
	w.lavaForce, w.lavaForceNext = w.lavaForceNext, w.lavaForce
}

func (w *World) processLavaVents() {
	if len(w.lavaVents) == 0 {
		return
	}
	active := w.lavaVents[:0]
	for _, vent := range w.lavaVents {
		if vent.ttl <= 0 {
			continue
		}
		idx := vent.idx
		if idx < 0 || idx >= len(w.groundNext) {
			continue
		}
		height := int(w.lavaHeightNext[idx]) + lavaFluxPerTick
		if height > lavaMaxHeight {
			height = lavaMaxHeight
		}
		w.lavaHeightNext[idx] = uint8(height)
		w.lavaTempNext[idx] = 1
		w.lavaDirNext[idx] = vent.dir
		w.lavaForceNext[idx] = height >= lavaOverflowHeight
		w.groundNext[idx] = GroundLava
		if idx < len(w.vegCurr) {
			w.vegCurr[idx] = VegetationNone
		}
		if idx < len(w.vegNext) {
			w.vegNext[idx] = VegetationNone
		}
		if idx < len(w.burnTTL) {
			w.burnTTL[idx] = 0
		}
		if idx < len(w.burnNext) {
			w.burnNext[idx] = 0
		}

		outIdx := vent.outIdx
		if outIdx >= 0 && outIdx < len(w.groundNext) {
			if w.groundNext[outIdx] != GroundLava {
				w.groundNext[outIdx] = GroundLava
			}
			if w.lavaHeightNext[outIdx] < 1 {
				w.lavaHeightNext[outIdx] = 1
			}
			w.lavaTempNext[outIdx] = 1
			w.lavaDirNext[outIdx] = vent.dir
			w.lavaForceNext[outIdx] = int(w.lavaHeightNext[outIdx]) >= lavaOverflowHeight
			w.lavaTipNext[outIdx] = true
			w.lavaAdvancedCells = append(w.lavaAdvancedCells, outIdx)
			if outIdx < len(w.vegCurr) {
				w.vegCurr[outIdx] = VegetationNone
			}
			if outIdx < len(w.vegNext) {
				w.vegNext[outIdx] = VegetationNone
			}
			if outIdx < len(w.burnTTL) {
				w.burnTTL[outIdx] = 0
			}
			if outIdx < len(w.burnNext) {
				w.burnNext[outIdx] = 0
			}
		}

		vent.ttl--
		if vent.ttl > 0 {
			active = append(active, vent)
		}
	}
	w.lavaVents = active
}

func (w *World) spawnLavaChild(cand lavaCandidate, temp float32) bool {
	nIdx := cand.idx
	if nIdx < 0 || nIdx >= len(w.groundNext) {
		return false
	}
	if w.groundCurr[nIdx] == GroundLava || w.groundNext[nIdx] == GroundLava {
		return false
	}
	ground := w.groundCurr[nIdx]
	if ground != GroundDirt && ground != GroundRock {
		return false
	}
	if temp < 0 {
		temp = 0
	}
	if temp > 1 {
		temp = 1
	}
	w.groundNext[nIdx] = GroundLava
	w.lavaHeightNext[nIdx] = 1
	w.lavaTempNext[nIdx] = temp
	w.lavaDirNext[nIdx] = cand.dir
	w.lavaForceNext[nIdx] = false
	w.lavaTipNext[nIdx] = true
	if nIdx < len(w.vegCurr) {
		w.vegCurr[nIdx] = VegetationNone
	}
	if nIdx < len(w.vegNext) {
		w.vegNext[nIdx] = VegetationNone
	}
	if nIdx < len(w.burnTTL) {
		w.burnTTL[nIdx] = 0
	}
	if nIdx < len(w.burnNext) {
		w.burnNext[nIdx] = 0
	}
	w.lavaAdvancedCells = append(w.lavaAdvancedCells, nIdx)
	return true
}

func (w *World) advanceLavaTip(idx int) bool {
	if idx < 0 || idx >= len(w.groundCurr) {
		return false
	}
	if w.groundCurr[idx] != GroundLava {
		return false
	}
	dir := w.lavaDir[idx]
	if dir < 0 {
		return false
	}
	height := int(w.lavaHeight[idx])
	if height <= 0 {
		return false
	}
	temp := w.lavaTemp[idx]
	forceAdvance := w.lavaForce[idx]
	chance := lavaBaseSpeed * float64(temp) / (1 + lavaSpeedAlpha*float64(height))
	if forceAdvance {
		chance = 1
	}
	if chance <= 0 {
		return false
	}
	if chance > 1 {
		chance = 1
	}
	if !forceAdvance && w.rng.Float64() >= chance {
		return false
	}

	x := idx % w.w
	y := idx / w.w
	elevHere := w.lavaElevation[idx]

	var candidates [8]lavaCandidate
	count := 0
	addCandidate := func(nIdx int, dirIdx int8) {
		if dirIdx < 0 || nIdx < 0 || nIdx >= len(w.groundNext) {
			return
		}
		if w.groundCurr[nIdx] == GroundLava || w.groundNext[nIdx] == GroundLava {
			return
		}
		ground := w.groundCurr[nIdx]
		if ground != GroundDirt && ground != GroundRock {
			return
		}
		for i := 0; i < count; i++ {
			if candidates[i].idx == nIdx {
				return
			}
		}
		slope := 0.0
		if nIdx < len(w.lavaElevation) {
			diff := int(elevHere) - int(w.lavaElevation[nIdx])
			if diff > 0 {
				slope = float64(diff)
			}
		}
		align := lavaDot(dirIdx, dir)
		if forceAdvance {
			align = 0
		}
		channel := 0.0
		if nIdx < len(w.lavaChannel) {
			channel = float64(w.lavaChannel[nIdx])
		}
		rain := 0.0
		if nIdx < len(w.rainCurr) {
			rain = float64(w.rainCurr[nIdx])
		}
		wall := 0.0
		if nIdx < len(w.lavaElevation) {
			diff := int(w.lavaElevation[nIdx]) - int(elevHere)
			if diff > 0 {
				wall = float64(diff)
			}
		}
		score := lavaSlopeWeight*slope + lavaAlignWeight*align + lavaChannelWeight*channel - lavaRainWeight*rain - lavaWallWeight*wall
		candidates[count] = lavaCandidate{idx: nIdx, dir: dirIdx, score: score}
		count++
	}

	forward := lavaDirections[dir]
	nx := x + forward.dx
	ny := y + forward.dy
	if nx >= 0 && nx < w.w && ny >= 0 && ny < w.h {
		addCandidate(ny*w.w+nx, dir)
	}
	leftDir := lavaLeft(dir)
	if leftDir >= 0 {
		dv := lavaDirections[leftDir]
		nx := x + dv.dx
		ny := y + dv.dy
		if nx >= 0 && nx < w.w && ny >= 0 && ny < w.h {
			addCandidate(ny*w.w+nx, leftDir)
		}
	}
	rightDir := lavaRight(dir)
	if rightDir >= 0 {
		dv := lavaDirections[rightDir]
		nx := x + dv.dx
		ny := y + dv.dy
		if nx >= 0 && nx < w.w && ny >= 0 && ny < w.h {
			addCandidate(ny*w.w+nx, rightDir)
		}
	}

	for dirIdx, dv := range lavaDirections {
		nx := x + dv.dx
		ny := y + dv.dy
		if nx < 0 || nx >= w.w || ny < 0 || ny >= w.h {
			continue
		}
		nIdx := ny*w.w + nx
		if nIdx < 0 || nIdx >= len(w.lavaElevation) {
			continue
		}
		if w.lavaElevation[nIdx] < elevHere {
			addCandidate(nIdx, int8(dirIdx))
		}
	}

	if count == 0 {
		return false
	}

	best := -1
	second := -1
	for i := 0; i < count; i++ {
		if best < 0 || candidates[i].score > candidates[best].score {
			if best >= 0 {
				second = best
			}
			best = i
		} else if (second < 0 || candidates[i].score > candidates[second].score) && candidates[i].idx != candidates[best].idx {
			second = i
		}
	}
	if best < 0 {
		return false
	}
	if candidates[best].score < lavaFlowThreshold {
		if !(forceAdvance && candidates[best].score >= 0) {
			return false
		}
	}

	childTemp := temp - 0.05
	if childTemp < 0 {
		childTemp = 0
	}

	advanced := w.spawnLavaChild(candidates[best], childTemp)
	if !advanced {
		return false
	}

	newHeight := height - 1
	if newHeight < 1 {
		newHeight = 1
	}
	w.lavaHeightNext[idx] = uint8(newHeight)
	w.lavaForceNext[idx] = newHeight >= lavaOverflowHeight

	threshold := lavaFlowThreshold - lavaSplitThresholdDrop
	if threshold < 0 {
		threshold = 0
	}
	if height >= lavaSplitMinHeight && w.rng.Float64() < lavaSplitChance && second >= 0 && candidates[second].idx != candidates[best].idx && candidates[second].score >= threshold {
		_ = w.spawnLavaChild(candidates[second], childTemp)
	}

	return true
}

func (w *World) poolFailedTips() {
	if len(w.lavaFailedTips) == 0 {
		return
	}
	for _, idx := range w.lavaFailedTips {
		if idx < 0 || idx >= len(w.groundCurr) {
			continue
		}
		if w.groundCurr[idx] != GroundLava {
			continue
		}
		x := idx % w.w
		y := idx / w.w
		elevHere := w.lavaElevation[idx]
		var options [8]int
		optionCount := 0
		for _, dv := range lavaDirections {
			nx := x + dv.dx
			ny := y + dv.dy
			if nx < 0 || nx >= w.w || ny < 0 || ny >= w.h {
				continue
			}
			nIdx := ny*w.w + nx
			if w.groundCurr[nIdx] == GroundLava || w.groundNext[nIdx] == GroundLava {
				continue
			}
			ground := w.groundCurr[nIdx]
			if ground != GroundDirt && ground != GroundRock {
				continue
			}
			if nIdx < len(w.lavaElevation) && w.lavaElevation[nIdx] > elevHere {
				continue
			}
			options[optionCount] = nIdx
			optionCount++
		}
		if optionCount > 0 {
			target := options[w.rng.Intn(optionCount)]
			temp := w.lavaTemp[idx] - 0.1
			if temp < 0 {
				temp = 0
			}
			w.groundNext[target] = GroundLava
			w.lavaHeightNext[target] = 1
			w.lavaTempNext[target] = temp
			w.lavaDirNext[target] = -1
			w.lavaForceNext[target] = false
			w.lavaTipNext[target] = false
			if target < len(w.vegCurr) {
				w.vegCurr[target] = VegetationNone
			}
			if target < len(w.vegNext) {
				w.vegNext[target] = VegetationNone
			}
			if target < len(w.burnTTL) {
				w.burnTTL[target] = 0
			}
			if target < len(w.burnNext) {
				w.burnNext[target] = 0
			}
		}

		newHeight := int(w.lavaHeight[idx]) + 1
		if newHeight > lavaMaxHeight {
			newHeight = lavaMaxHeight
		}
		w.lavaHeightNext[idx] = uint8(newHeight)
		w.lavaForceNext[idx] = newHeight >= lavaOverflowHeight
	}
}

func (w *World) feedCalderaReservoir() {
	res := &w.lavaReservoir
	if res == nil || res.ticks <= 0 || len(res.cells) == 0 {
		return
	}
	for _, idx := range res.cells {
		if idx < 0 || idx >= len(w.groundNext) {
			continue
		}
		if w.groundNext[idx] != GroundLava {
			continue
		}
		height := int(w.lavaHeightNext[idx])
		if height < lavaReservoirTargetHeight {
			height = lavaReservoirTargetHeight
			if height > lavaMaxHeight {
				height = lavaMaxHeight
			}
			w.lavaHeightNext[idx] = uint8(height)
		}
		if height >= lavaOverflowHeight {
			w.lavaForceNext[idx] = true
		}
		temp := float64(w.lavaTempNext[idx])
		if temp < lavaReservoirMinTemp {
			temp = lavaReservoirMinTemp
		}
		if temp > 1 {
			temp = 1
		}
		w.lavaTempNext[idx] = float32(temp)
		w.lavaDirNext[idx] = -1
		w.lavaTipNext[idx] = false
	}
	res.ticks--
	if res.ticks <= 0 {
		res.reset()
	}
}

func (w *World) coolLavaCells() {
	total := w.w * w.h
	for idx := 0; idx < total; idx++ {
		if w.groundNext[idx] != GroundLava {
			w.lavaHeightNext[idx] = 0
			w.lavaTempNext[idx] = 0
			w.lavaDirNext[idx] = -1
			w.lavaForceNext[idx] = false
			w.lavaTipNext[idx] = false
			continue
		}

		height := int(w.lavaHeightNext[idx])
		if height <= 0 {
			height = 1
			w.lavaHeightNext[idx] = 1
		}

		x := idx % w.w
		y := idx / w.w
		neighbors := 0
		for _, dv := range lavaDirections {
			nx := x + dv.dx
			ny := y + dv.dy
			if nx < 0 || nx >= w.w || ny < 0 || ny >= w.h {
				continue
			}
			nIdx := ny*w.w + nx
			if w.groundNext[nIdx] == GroundLava {
				neighbors++
			}
		}

		edgeFactor := 1 - float64(neighbors)/float64(len(lavaDirections))
		if edgeFactor < 0 {
			edgeFactor = 0
		}
		rain := 0.0
		if idx < len(w.rainCurr) {
			rain = float64(w.rainCurr[idx])
		}
		cool := lavaCoolBase + lavaCoolEdge*edgeFactor + lavaCoolRain*rain + lavaCoolThick*lavaSigmoid(float64(height-2))
		if w.lavaDirNext[idx] >= 0 {
			cool += lavaCoolFlowBonus
		}
		temp := float64(w.lavaTempNext[idx]) - cool
		if temp < 0 {
			temp = 0
		}
		w.lavaTempNext[idx] = float32(temp)

		if temp <= lavaCrustThreshold {
			if height > 1 {
				height--
				if height < 1 {
					height = 1
				}
				w.lavaHeightNext[idx] = uint8(height)
				if temp > lavaReheatCap {
					temp = lavaReheatCap
				}
				w.lavaTempNext[idx] = float32(temp)
			} else {
				w.groundNext[idx] = GroundRock
				w.lavaHeightNext[idx] = 0
				w.lavaTempNext[idx] = 0
				w.lavaDirNext[idx] = -1
				w.lavaForceNext[idx] = false
				w.lavaTipNext[idx] = false
				continue
			}
		}

		w.lavaForceNext[idx] = height >= lavaOverflowHeight
	}
}

func (w *World) reinforceLavaChannels() {
	for _, idx := range w.lavaAdvancedCells {
		if idx < 0 || idx >= len(w.lavaChannel) {
			continue
		}
		value := w.lavaChannel[idx] + float32(lavaChannelGrow)
		if value > 1 {
			value = 1
		}
		w.lavaChannel[idx] = value
	}

	decay := float32(1 - lavaChannelDecay)
	for i := range w.lavaChannel {
		w.lavaChannel[i] *= decay
		if w.lavaChannel[i] < 0 {
			w.lavaChannel[i] = 0
		}
	}
}

func (w *World) detectLavaTips() {
	total := w.w * w.h
	for idx := 0; idx < total; idx++ {
		if w.groundNext[idx] != GroundLava {
			w.lavaTipNext[idx] = false
			w.lavaForceNext[idx] = false
			continue
		}
		w.lavaForceNext[idx] = w.lavaHeightNext[idx] >= lavaOverflowHeight
		dir := w.lavaDirNext[idx]
		if dir < 0 {
			w.lavaTipNext[idx] = false
			continue
		}
		if w.lavaTempNext[idx] <= float32(lavaTipTemperatureMin) {
			w.lavaTipNext[idx] = false
			continue
		}
		x := idx % w.w
		y := idx / w.w
		neighbors := 0
		for _, dv := range lavaDirections {
			nx := x + dv.dx
			ny := y + dv.dy
			if nx < 0 || nx >= w.w || ny < 0 || ny >= w.h {
				continue
			}
			nIdx := ny*w.w + nx
			if w.groundNext[nIdx] == GroundLava {
				neighbors++
			}
		}
		w.lavaTipNext[idx] = neighbors <= 2
	}
}

func (w *World) applyFire() {
	total := w.w * w.h
	if total == 0 || len(w.burnTTL) != total || len(w.burnNext) != total {
		return
	}

	for i := range w.burnNext {
		w.burnNext[i] = 0
	}

	baseTTL := w.cfg.Params.BurnTTL
	if baseTTL <= 0 {
		baseTTL = 1
	}
	spreadChance := w.cfg.Params.FireSpreadChance
	if spreadChance < 0 {
		spreadChance = 0
	}
	rainDampen := w.cfg.Params.FireRainSpreadDampen
	rainExtinguish := w.cfg.Params.FireRainExtinguishChance
	lavaIgnite := w.cfg.Params.FireLavaIgniteChance
	if lavaIgnite < 0 {
		lavaIgnite = 0
	}

	rainModifier := func(value float64) float64 {
		if rainDampen <= 0 || value <= 0 {
			return 1
		}
		modifier := 1 - rainDampen*value
		if modifier < 0 {
			return 0
		}
		if modifier > 1 {
			return 1
		}
		return modifier
	}

	for y := 0; y < w.h; y++ {
		for x := 0; x < w.w; x++ {
			idx := y*w.w + x
			ttl := int(w.burnTTL[idx])
			if ttl <= 0 {
				continue
			}

			rainHere := 0.0
			if idx < len(w.rainCurr) {
				rainHere = float64(w.rainCurr[idx])
			}

			extinguished := false
			if rainExtinguish > 0 && rainHere > 0 {
				chance := rainExtinguish * rainHere
				if chance < 0 {
					chance = 0
				}
				if chance > 1 {
					chance = 1
				}
				if w.rng.Float64() < chance {
					extinguished = true
				}
			}

			newTTL := ttl - 1
			if newTTL < 0 {
				newTTL = 0
			}
			if extinguished {
				newTTL = 0
			}

			if newTTL > 0 {
				if newTTL > 255 {
					newTTL = 255
				}
				w.burnNext[idx] = uint8(newTTL)
				if idx < len(w.display) {
					w.display[idx] = 1
				}
			} else {
				w.vegCurr[idx] = VegetationNone
				if idx < len(w.display) {
					base := uint8(0)
					if idx < len(w.groundCurr) {
						base = uint8(w.groundCurr[idx])
					}
					w.display[idx] = base
				}
			}

			if spreadChance <= 0 {
				continue
			}

			for dy := -1; dy <= 1; dy++ {
				ny := y + dy
				if ny < 0 || ny >= w.h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					nx := x + dx
					if nx < 0 || nx >= w.w {
						continue
					}
					if dx == 0 && dy == 0 {
						continue
					}
					nIdx := ny*w.w + nx
					if w.vegCurr[nIdx] == VegetationNone {
						continue
					}
					if int(w.burnTTL[nIdx]) > 0 {
						continue
					}

					chance := spreadChance
					if nIdx < len(w.rainCurr) {
						chance *= rainModifier(float64(w.rainCurr[nIdx]))
					}
					if chance <= 0 {
						continue
					}
					if chance > 1 {
						chance = 1
					}
					if w.rng.Float64() >= chance {
						continue
					}

					ttlVal := baseTTL
					if ttlVal <= 0 {
						ttlVal = 1
					}
					if existing := int(w.burnNext[nIdx]); existing > ttlVal {
						ttlVal = existing
					}
					if ttlVal > 255 {
						ttlVal = 255
					}
					w.burnNext[nIdx] = uint8(ttlVal)
					if nIdx < len(w.display) {
						w.display[nIdx] = 1
					}
				}
			}
		}
	}

	if lavaIgnite > 0 {
		for y := 0; y < w.h; y++ {
			for x := 0; x < w.w; x++ {
				idx := y*w.w + x
				if idx < 0 || idx >= len(w.groundCurr) {
					continue
				}
				if w.groundCurr[idx] != GroundLava {
					continue
				}
				if idx < len(w.lavaHeight) && w.lavaHeight[idx] == 0 {
					continue
				}

				for dy := -1; dy <= 1; dy++ {
					ny := y + dy
					if ny < 0 || ny >= w.h {
						continue
					}
					for dx := -1; dx <= 1; dx++ {
						nx := x + dx
						if nx < 0 || nx >= w.w {
							continue
						}
						if dx == 0 && dy == 0 {
							continue
						}
						nIdx := ny*w.w + nx
						if nIdx < 0 || nIdx >= len(w.vegCurr) {
							continue
						}
						if w.vegCurr[nIdx] == VegetationNone {
							continue
						}
						if int(w.burnTTL[nIdx]) > 0 {
							continue
						}
						if nIdx < len(w.burnNext) && int(w.burnNext[nIdx]) > 0 {
							continue
						}

						chance := lavaIgnite
						if nIdx < len(w.rainCurr) {
							chance *= rainModifier(float64(w.rainCurr[nIdx]))
						}
						if chance <= 0 {
							continue
						}
						if chance > 1 {
							chance = 1
						}
						if w.rng.Float64() >= chance {
							continue
						}

						ttlVal := baseTTL
						if ttlVal <= 0 {
							ttlVal = 1
						}
						if ttlVal > 255 {
							ttlVal = 255
						}
						if nIdx < len(w.burnNext) {
							w.burnNext[nIdx] = uint8(ttlVal)
						}
						if nIdx < len(w.display) {
							w.display[nIdx] = 1
						}
					}
				}
			}
		}
	}

	w.burnTTL, w.burnNext = w.burnNext, w.burnTTL
}

// IgniteAt manually starts a burn at the provided coordinates when vegetation is present.
func (w *World) IgniteAt(x, y int) {
	if x < 0 || y < 0 || x >= w.w || y >= w.h {
		return
	}
	idx := y*w.w + x
	if idx < 0 || idx >= len(w.vegCurr) {
		return
	}
	if w.vegCurr[idx] == VegetationNone {
		return
	}
	base := w.cfg.Params.BurnTTL
	if base <= 0 {
		base = 1
	}
	ttl := base
	if existing := int(w.burnTTL[idx]); existing > 0 {
		ttl += existing
	}
	if ttl > 255 {
		ttl = 255
	}
	w.burnTTL[idx] = uint8(ttl)
	w.rebuildDisplay()
}

// Metrics exposes the latest vegetation telemetry.
func (w *World) Metrics() VegetationMetrics {
	m := w.metrics
	if len(m.ClusterHistogram) > 0 {
		m.ClusterHistogram = append([]int(nil), m.ClusterHistogram...)
	}
	return m
}

// EnvironmentSummary returns environment telemetry derived from the current buffers.
func (w *World) EnvironmentSummary() EnvironmentMetrics {
	total := w.w * w.h
	if total == 0 {
		return EnvironmentMetrics{}
	}

	var metrics EnvironmentMetrics
	metrics.TotalTiles = total
	metrics.ActiveRainRegions = len(w.rainRegions)

	var rainSum float64
	for i := 0; i < total; i++ {
		if i < len(w.groundCurr) {
			switch w.groundCurr[i] {
			case GroundDirt:
				metrics.DirtTiles++
			case GroundRock:
				metrics.RockTiles++
			case GroundMountain:
				metrics.MountainTiles++
			case GroundLava:
				metrics.LavaTiles++
			}
		}

		if i < len(w.burnTTL) && w.burnTTL[i] > 0 {
			metrics.BurningTiles++
		}

		if i < len(w.rainCurr) {
			value := float64(w.rainCurr[i])
			rainSum += value
			if value > 0 {
				metrics.RainCoverage++
				if value > metrics.RainMax {
					metrics.RainMax = value
				}
			}
		}
	}

	metrics.RainMean = rainSum / float64(total)
	return metrics
}

func (w *World) updateMetrics(buffer []Vegetation) {
	total := w.w * w.h
	if total == 0 {
		w.metrics = VegetationMetrics{}
		return
	}

	var m VegetationMetrics

	for i := 0; i < total; i++ {
		switch buffer[i] {
		case VegetationGrass:
			m.GrassTiles++
		case VegetationShrub:
			m.ShrubTiles++
		case VegetationTree:
			m.TreeTiles++
		}
	}
	m.TotalVegetated = m.GrassTiles + m.ShrubTiles + m.TreeTiles

	visited := make([]bool, total)
	componentCounts := make(map[int]int)
	maxSize := 0

	for idx := 0; idx < total; idx++ {
		if visited[idx] {
			continue
		}
		if buffer[idx] == VegetationNone {
			continue
		}

		size := 0
		stack := []int{idx}
		visited[idx] = true
		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++

			cx := current % w.w
			cy := current / w.w
			for dy := -1; dy <= 1; dy++ {
				ny := cy + dy
				if ny < 0 || ny >= w.h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					nx := cx + dx
					if nx < 0 || nx >= w.w {
						continue
					}
					if dx == 0 && dy == 0 {
						continue
					}
					nIdx := ny*w.w + nx
					if visited[nIdx] {
						continue
					}
					if buffer[nIdx] == VegetationNone {
						continue
					}
					visited[nIdx] = true
					stack = append(stack, nIdx)
				}
			}
		}

		componentCounts[size]++
		if size > maxSize {
			maxSize = size
		}
	}

	if maxSize > 0 {
		m.ClusterHistogram = make([]int, maxSize+1)
		for size, count := range componentCounts {
			m.ClusterHistogram[size] = count
		}
	}

	w.metrics = m
}

func (w *World) mooreNeighborCounts() ([]uint8, []uint8) {
	total := w.w * w.h
	grassCounts := make([]uint8, total)
	shrubCounts := make([]uint8, total)
	if total == 0 {
		return grassCounts, shrubCounts
	}

	for y := 0; y < w.h; y++ {
		for x := 0; x < w.w; x++ {
			idx := y*w.w + x
			val := w.vegCurr[idx]
			if val != VegetationGrass && val != VegetationShrub {
				continue
			}

			for dy := -1; dy <= 1; dy++ {
				ny := y + dy
				if ny < 0 || ny >= w.h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					nx := x + dx
					if nx < 0 || nx >= w.w {
						continue
					}
					if dx == 0 && dy == 0 {
						continue
					}
					nIdx := ny*w.w + nx
					if val == VegetationGrass {
						grassCounts[nIdx]++
					} else {
						shrubCounts[nIdx]++
					}
				}
			}
		}
	}

	return grassCounts, shrubCounts
}

func (w *World) sprinkleRock() {
	if w.cfg.Params.RockChance <= 0 {
		return
	}
	total := w.w * w.h
	for i := 0; i < total; i++ {
		if w.rng.Float64() < w.cfg.Params.RockChance {
			w.groundCurr[i] = GroundRock
			w.display[i] = uint8(GroundRock)
		} else {
			w.display[i] = uint8(w.groundCurr[i])
		}
	}
}

func (w *World) seedGrassPatches() {
	count := w.cfg.Params.GrassPatchCount
	if count <= 0 {
		return
	}
	minR := w.cfg.Params.GrassPatchRadiusMin
	maxR := w.cfg.Params.GrassPatchRadiusMax
	if minR < 0 {
		minR = 0
	}
	if maxR < minR {
		maxR = minR
	}
	den := w.cfg.Params.GrassPatchDensity
	if den <= 0 {
		den = 1
	}
	for p := 0; p < count; p++ {
		x := w.rng.Intn(w.w)
		y := w.rng.Intn(w.h)
		radius := minR
		if maxR > minR {
			radius += w.rng.Intn(maxR - minR + 1)
		}
		r2 := radius * radius
		for dy := -radius; dy <= radius; dy++ {
			yp := y + dy
			if yp < 0 || yp >= w.h {
				continue
			}
			for dx := -radius; dx <= radius; dx++ {
				xp := x + dx
				if xp < 0 || xp >= w.w {
					continue
				}
				if dx*dx+dy*dy > r2 {
					continue
				}
				if w.rng.Float64() > den {
					continue
				}
				idx := yp*w.w + xp
				w.vegCurr[idx] = VegetationGrass
			}
		}
	}
}

func loadTectonicMap(w, h int) []float32 {
	total := w * h
	if total <= 0 {
		return nil
	}
	vals := make([]float32, total)
	if w == 0 {
		return vals
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			vals[idx] = float32(0.5 + 0.5*math.Sin(float64(x)/float64(w)*math.Pi))
		}
	}
	return vals
}

func init() {
	core.Register("ecology", func(cfg map[string]string) core.Sim {
		c := FromMap(cfg)
		return NewWithConfig(c)
	})
}
