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

	groundCurr  []Ground
	groundNext  []Ground
	vegCurr     []Vegetation
	vegNext     []Vegetation
	lavaLife    []uint8
	lavaNext    []uint8
	burnTTL     []uint8
	burnNext    []uint8
	rainCurr    []float32
	rainNext    []float32
	rainScratch []float32
	volCurr     []float32
	volNext     []float32
	tectonic    []float32
	display     []uint8

	rng *rand.Rand

	metrics VegetationMetrics

	rainRegions          []rainRegion
	volcanoRegions       []volcanoProtoRegion
	expiredVolcanoProtos []volcanoProtoRegion
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
	cx, cy            float64
	radiusX           float64
	radiusY           float64
	strength          float64
	baseStrength      float64
	strengthVariation float64
	ttl               int
	maxTTL            int
	age               int
	vx, vy            float64
	threshold         float64
	falloff           float64
	noiseScale        float64
	noiseStretchX     float64
	noiseStretchY     float64
	noiseSeed         int64
	angle             float64
	preset            rainPreset
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
		floatControl("grass_spread_chance", "Grass spread chance", 0.05, 0, 1),
		floatControl("shrub_growth_chance", "Shrub growth chance", 0.01, 0, 0.2),
		floatControl("fire_spread_chance", "Fire spread chance", 0.05, 0, 1),
		floatControl("fire_rain_spread_dampen", "Rain dampen factor", 0.05, 0, 1),
		floatControl("rain_spawn_chance", "Rain spawn chance", 0.01, 0, 0.2),
		floatControl("rain_strength_max", "Rain strength max", 0.05, 0, 1),
		floatControl("lava_spread_chance", "Lava spread chance", 0.05, 0, 1),
		floatControl("volcano_proto_spawn_chance", "Volcano proto spawn chance", 0.01, 0, 0.5),
		floatControl("volcano_eruption_chance_base", "Volcano eruption chance", 0.01, 0, 1),
		intControl("lava_life_min", "Lava life min", 1, 1, 90),
		intControl("lava_life_max", "Lava life max", 1, 1, 120),
		intControl("burn_ttl", "Burn duration", 1, 1, 10),
	}
}

// SetIntParameter allows HUD interactions to update integer ecology parameters.
func (w *World) SetIntParameter(key string, value int) bool {
	if w == nil {
		return false
	}
	switch key {
	case "lava_life_min":
		clamped := clampInt(value, 1, 90)
		if clamped > w.cfg.Params.LavaLifeMax {
			w.cfg.Params.LavaLifeMax = clamped
		}
		w.cfg.Params.LavaLifeMin = clamped
		return true
	case "lava_life_max":
		clamped := clampInt(value, 1, 120)
		if clamped < w.cfg.Params.LavaLifeMin {
			clamped = w.cfg.Params.LavaLifeMin
		}
		w.cfg.Params.LavaLifeMax = clamped
		return true
	case "burn_ttl":
		w.cfg.Params.BurnTTL = clampInt(value, 1, 10)
		return true
	default:
		return false
	}
}

// SetFloatParameter allows HUD interactions to update float ecology parameters.
func (w *World) SetFloatParameter(key string, value float64) bool {
	if w == nil {
		return false
	}
	switch key {
	case "grass_spread_chance":
		w.cfg.Params.GrassSpreadChance = clampFloat(value, 0, 1)
		return true
	case "shrub_growth_chance":
		w.cfg.Params.ShrubGrowthChance = clampFloat(value, 0, 0.2)
		return true
	case "fire_spread_chance":
		w.cfg.Params.FireSpreadChance = clampFloat(value, 0, 1)
		return true
	case "fire_rain_spread_dampen":
		w.cfg.Params.FireRainSpreadDampen = clampFloat(value, 0, 1)
		return true
	case "rain_spawn_chance":
		w.cfg.Params.RainSpawnChance = clampFloat(value, 0, 0.2)
		return true
	case "rain_strength_max":
		clamped := clampFloat(value, 0, 1)
		if clamped < w.cfg.Params.RainStrengthMin {
			clamped = w.cfg.Params.RainStrengthMin
		}
		w.cfg.Params.RainStrengthMax = clamped
		return true
	case "lava_spread_chance":
		w.cfg.Params.LavaSpreadChance = clampFloat(value, 0, 1)
		return true
	case "volcano_proto_spawn_chance":
		w.cfg.Params.VolcanoProtoSpawnChance = clampFloat(value, 0, 0.5)
		return true
	case "volcano_eruption_chance_base":
		w.cfg.Params.VolcanoEruptionChanceBase = clampFloat(value, 0, 1)
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
		cfg:         cfg,
		w:           cfg.Width,
		h:           cfg.Height,
		groundCurr:  make([]Ground, total),
		groundNext:  make([]Ground, total),
		vegCurr:     make([]Vegetation, total),
		vegNext:     make([]Vegetation, total),
		lavaLife:    make([]uint8, total),
		lavaNext:    make([]uint8, total),
		burnTTL:     make([]uint8, total),
		burnNext:    make([]uint8, total),
		rainCurr:    make([]float32, total),
		rainNext:    make([]float32, total),
		rainScratch: make([]float32, total),
		volCurr:     make([]float32, total),
		volNext:     make([]float32, total),
		tectonic:    loadTectonicMap(cfg.Width, cfg.Height),
		display:     make([]uint8, total),
		rng:         rand.New(rand.NewSource(cfg.Seed)),
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
	total := w.w * w.h
	for i := 0; i < total; i++ {
		w.groundCurr[i] = GroundDirt
		w.groundNext[i] = GroundDirt
		w.vegCurr[i] = VegetationNone
		w.vegNext[i] = VegetationNone
		w.lavaLife[i] = 0
		w.lavaNext[i] = 0
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
}

func floatControl(key, label string, step, min, max float64) core.ParameterControl {
	return core.ParameterControl{
		Key:    key,
		Label:  label,
		Type:   core.ParamTypeFloat,
		Step:   step,
		Min:    min,
		Max:    max,
		HasMin: true,
		HasMax: true,
	}
}

func intControl(key, label string, step float64, min, max int) core.ParameterControl {
	return core.ParameterControl{
		Key:    key,
		Label:  label,
		Type:   core.ParamTypeInt,
		Step:   step,
		Min:    float64(min),
		Max:    float64(max),
		HasMin: true,
		HasMax: true,
	}
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
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

// Step advances the simulation by applying the vegetation succession rules once.
func (w *World) Step() {
	if w.w == 0 || w.h == 0 {
		return
	}

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

	active := w.rainRegions[:0]
	for i := range w.rainRegions {
		region := w.rainRegions[i]
		if region.ttl <= 0 {
			continue
		}
		region = w.advanceRainRegion(region)
		active = append(active, region)
	}

	w.rainRegions = active

	var absorbed []bool
	if len(w.rainRegions) > 1 {
		absorbed = make([]bool, len(w.rainRegions))
		for i := 0; i < len(w.rainRegions); i++ {
			for j := i + 1; j < len(w.rainRegions); j++ {
				overlap := rainOverlapRatio(w.rainRegions[i], w.rainRegions[j])
				if overlap <= 0.2 {
					continue
				}
				larger := i
				smaller := j
				if w.rainRegions[j].area() > w.rainRegions[i].area() {
					larger, smaller = j, i
				}
				boosted := clamp01(w.rainRegions[larger].baseStrength + 0.1)
				w.rainRegions[larger].baseStrength = boosted
				if w.rainRegions[larger].strength < boosted {
					w.rainRegions[larger].strength = boosted
				}
				absorbed[smaller] = true
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
			if len(absorbed) > 0 && absorbed[i] {
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
		falloff = 1.4
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
			n := fbmNoise2D(nx, ny, 3, 0.5, 1.9, region.noiseSeed)

			distX := rx * invRadiusX
			distY := ry * invRadiusY
			radial := math.Sqrt(distX*distX + distY*distY)
			if radial > 1 {
				continue
			}

			// SOFT CLOUD MASK
			c := smoothstep(threshold-0.08, threshold+0.08, n)
			if c <= 0 {
				continue
			}

			// RADIAL CORE FLOOR (prevents central holes)
			if radial < 0.35 {
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

func (w *World) advanceRainRegion(region rainRegion) rainRegion {
	region.age++

	targetVX, targetVY := w.windVector(region.cx, region.cy, region.noiseSeed)
	region.vx += (targetVX - region.vx) * 0.05
	region.vy += (targetVY - region.vy) * 0.05

	jitterX, jitterY := correlatedJitter(region.noiseSeed, region.age)
	region.vx += jitterX * 0.05
	region.vy += jitterY * 0.05

	region.cx += region.vx
	region.cy += region.vy

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
	region.strength = clamp01(region.baseStrength * factor)

	if region.preset != rainPresetPuffy {
		speed := math.Hypot(region.vx, region.vy)
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

	radius := 2 // was 1; better at sealing noise-made pinholes
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
					v := w.rainNext[ny*w.w+nx]
					if v > maxVal {
						maxVal = v
					}
				}
			}
			w.rainScratch[y*w.w+x] = maxVal
		}
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
					v := w.rainScratch[ny*w.w+nx]
					if !hasVal || v < minVal {
						minVal = v
						hasVal = true
					}
				}
			}
			if !hasVal {
				minVal = 0
			}
			if minVal < 0.02 {
				minVal = 0
			}
			w.rainNext[y*w.w+x] = minVal
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

	threshold := 0.35 + w.rng.Float64()*0.1
	falloff := 1.4
	noiseScale := 0.08 + w.rng.Float64()*0.01
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
		falloff = 1.3 + w.rng.Float64()*0.2
	case presetRoll < 0.85:
		preset = rainPresetStratus
		radiusX = baseRadius * (1.1 + w.rng.Float64()*0.4)
		radiusY = baseRadius * 0.6
		stretchY = 0.6
		noiseScale = 0.065 + w.rng.Float64()*0.015
		falloff = 1.2 + w.rng.Float64()*0.2
	default:
		preset = rainPresetSquall
		radiusX = 40 + w.rng.Float64()*20
		radiusY = 10 + w.rng.Float64()*6
		if radiusX > float64(params.RainRadiusMax)*1.5 {
			radiusX = float64(params.RainRadiusMax) * 1.5
		}
		stretchX = 1.2
		stretchY = 0.8
		noiseScale = 0.07 + w.rng.Float64()*0.02
		falloff = 1.1 + w.rng.Float64()*0.2
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
	vx, vy := w.windVector(cx, cy, seed)

	strengthVariation := 0.1 + w.rng.Float64()*0.1

	region := rainRegion{
		cx:                cx,
		cy:                cy,
		radiusX:           radiusX,
		radiusY:           radiusY,
		baseStrength:      baseStrength,
		strength:          baseStrength,
		strengthVariation: strengthVariation,
		ttl:               ttl,
		maxTTL:            ttl,
		vx:                vx,
		vy:                vy,
		threshold:         threshold,
		falloff:           falloff,
		noiseScale:        noiseScale,
		noiseStretchX:     stretchX,
		noiseStretchY:     stretchY,
		noiseSeed:         seed,
		preset:            preset,
	}

	if preset != rainPresetPuffy {
		speed := math.Hypot(vx, vy)
		if speed > 1e-3 {
			region.angle = math.Atan2(vy, vx)
		}
	}

	return region
}

func (w *World) windVector(x, y float64, seed int64) (float64, float64) {
	scale := 0.01
	sx := fbmNoise2D(x*scale, y*scale, 3, 0.5, 1.8, seed+17)
	sy := fbmNoise2D((x+300)*scale, (y-300)*scale, 3, 0.5, 1.8, seed+53)
	vx := (sx - 0.5) * 0.6
	vy := (sy - 0.5) * 0.6
	return vx, vy
}

func correlatedJitter(seed int64, age int) (float64, float64) {
	hx := hash2D(int64(age), 0, seed)
	hy := hash2D(int64(age), 1, seed^0x517cc1b727220a95)
	jx := (float64(hx)/float64(math.MaxUint32) - 0.5) * 0.3
	jy := (float64(hy)/float64(math.MaxUint32) - 0.5) * 0.3
	return jx, jy
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
	if len(w.lavaLife) != total {
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

	minLife := w.cfg.Params.LavaLifeMin
	maxLife := w.cfg.Params.LavaLifeMax
	if minLife <= 0 {
		minLife = 1
	}
	if maxLife < minLife {
		maxLife = minLife
	}

	const coreScale = 0.35
	const rimScale = 0.9
	const speckChance = 0.08

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

		w.eruptRegion(region, minLife, maxLife, coreScale, rimScale, speckChance)
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

func (w *World) eruptRegion(region volcanoProtoRegion, minLife, maxLife int, coreScale, rimScale, speckChance float64) {
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

	coreRadius := region.radius * coreScale
	rimRadius := region.radius * rimScale
	if rimRadius < coreRadius {
		rimRadius = coreRadius
	}

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
				w.placeLava(idx, minLife, maxLife)
			case dist <= rimRadius:
				if w.groundCurr[idx] == GroundRock {
					w.groundCurr[idx] = GroundMountain
					if idx < len(w.display) {
						w.display[idx] = uint8(GroundMountain)
					}
				}
				if idx < len(w.lavaLife) {
					w.lavaLife[idx] = 0
				}
			default:
				if speckChance <= 0 {
					continue
				}
				if w.rng.Float64() >= speckChance {
					continue
				}
				w.placeLava(idx, minLife, maxLife)
			}
		}
	}
}

func (w *World) placeLava(idx int, minLife, maxLife int) {
	if idx < 0 || idx >= len(w.groundCurr) {
		return
	}

	life := minLife
	if maxLife > minLife {
		life += w.rng.Intn(maxLife - minLife + 1)
	}
	if life < 1 {
		life = 1
	}
	if life > 255 {
		life = 255
	}

	w.groundCurr[idx] = GroundLava
	if idx < len(w.lavaLife) {
		w.lavaLife[idx] = uint8(life)
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
	if len(w.lavaLife) != total || len(w.lavaNext) != total {
		return
	}

	for i := 0; i < total; i++ {
		w.groundNext[i] = w.groundCurr[i]
		w.lavaNext[i] = 0
	}

	spreadChance := w.cfg.Params.LavaSpreadChance
	if spreadChance < 0 {
		spreadChance = 0
	}
	if spreadChance > 1 {
		spreadChance = 1
	}

	minLife := w.cfg.Params.LavaLifeMin
	maxLife := w.cfg.Params.LavaLifeMax
	if minLife <= 0 {
		minLife = 1
	}
	if maxLife < minLife {
		maxLife = minLife
	}

	spreadFloor := w.cfg.Params.LavaSpreadMaskFloor
	if spreadFloor < 0 {
		spreadFloor = 0
	}
	if spreadFloor > 1 {
		spreadFloor = 1
	}
	coolingExtra := w.cfg.Params.LavaCoolingExtra
	if coolingExtra < 0 {
		coolingExtra = 0
	}

	volMaskAt := func(idx int) float64 {
		if idx < 0 || idx >= len(w.volCurr) {
			return 0
		}
		val := float64(w.volCurr[idx])
		if val < 0 {
			return 0
		}
		if val > 1 {
			return 1
		}
		return val
	}

	rainAt := func(idx int) float64 {
		if idx < 0 || idx >= len(w.rainCurr) {
			return 0
		}
		val := float64(w.rainCurr[idx])
		if val < 0 {
			return 0
		}
		if val > 1 {
			return 1
		}
		return val
	}

	const rainCoolingScale = 8.0

	for y := 0; y < w.h; y++ {
		for x := 0; x < w.w; x++ {
			idx := y*w.w + x
			if w.groundCurr[idx] != GroundLava {
				continue
			}

			volMaskHere := volMaskAt(idx)
			rainHere := rainAt(idx)
			cooling := 1
			if coolingExtra > 0 {
				coolingDelta := int(math.Round((1 - volMaskHere) * coolingExtra))
				if coolingDelta > 0 {
					cooling += coolingDelta
				}
			}
			if rainHere > 0 {
				rainCooling := int(math.Round(rainHere * rainCoolingScale))
				if rainCooling > 0 {
					cooling += rainCooling
				}
			}
			if cooling < 1 {
				cooling = 1
			}

			life := int(w.lavaLife[idx]) - cooling
			if life > 0 {
				if life > 255 {
					life = 255
				}
				w.groundNext[idx] = GroundLava
				w.lavaNext[idx] = uint8(life)
				if idx < len(w.display) {
					w.display[idx] = uint8(GroundLava)
				}
			} else {
				w.groundNext[idx] = GroundRock
				w.lavaNext[idx] = 0
				if idx < len(w.display) {
					w.display[idx] = uint8(GroundRock)
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
					if w.groundCurr[nIdx] == GroundLava {
						continue
					}
					ground := w.groundCurr[nIdx]
					if ground != GroundDirt && ground != GroundRock {
						continue
					}
					heat := volMaskHere
					if targetMask := volMaskAt(nIdx); targetMask > heat {
						heat = targetMask
					}
					if heat < spreadFloor {
						heat = spreadFloor
					} else if heat > 1 {
						heat = 1
					}
					effectiveSpread := spreadChance * heat
					if effectiveSpread > 1 {
						effectiveSpread = 1
					}
					if w.rng.Float64() >= effectiveSpread {
						continue
					}

					lifeVal := minLife
					if maxLife > minLife {
						lifeVal += w.rng.Intn(maxLife - minLife + 1)
					}
					if lifeVal < 1 {
						lifeVal = 1
					}
					if lifeVal > 255 {
						lifeVal = 255
					}
					if existing := int(w.lavaNext[nIdx]); existing > lifeVal {
						lifeVal = existing
					}

					w.groundNext[nIdx] = GroundLava
					w.lavaNext[nIdx] = uint8(lifeVal)
					if nIdx < len(w.display) {
						w.display[nIdx] = uint8(GroundLava)
					}
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
				}
			}
		}
	}

	w.groundCurr, w.groundNext = w.groundNext, w.groundCurr
	w.lavaLife, w.lavaNext = w.lavaNext, w.lavaLife
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
				if idx < len(w.lavaLife) && w.lavaLife[idx] == 0 {
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
