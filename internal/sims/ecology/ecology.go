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

	groundCurr []Ground
	groundNext []Ground
	vegCurr    []Vegetation
	vegNext    []Vegetation
	lavaLife   []uint8
	lavaNext   []uint8
	burnTTL    []uint8
	burnNext   []uint8
	rainCurr   []float32
	rainNext   []float32
	volCurr    []float32
	volNext    []float32
	tectonic   []float32
	display    []uint8

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
	cx, cy   float64
	radius   float64
	strength float64
	ttl      int
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
		cfg:        cfg,
		w:          cfg.Width,
		h:          cfg.Height,
		groundCurr: make([]Ground, total),
		groundNext: make([]Ground, total),
		vegCurr:    make([]Vegetation, total),
		vegNext:    make([]Vegetation, total),
		lavaLife:   make([]uint8, total),
		lavaNext:   make([]uint8, total),
		burnTTL:    make([]uint8, total),
		burnNext:   make([]uint8, total),
		rainCurr:   make([]float32, total),
		rainNext:   make([]float32, total),
		volCurr:    make([]float32, total),
		volNext:    make([]float32, total),
		tectonic:   loadTectonicMap(cfg.Width, cfg.Height),
		display:    make([]uint8, total),
		rng:        rand.New(rand.NewSource(cfg.Seed)),
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

	w.rainRegions = w.rainRegions[:0]
	w.volcanoRegions = w.volcanoRegions[:0]
	w.expiredVolcanoProtos = w.expiredVolcanoProtos[:0]
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
}

func (w *World) updateRainMask() {
	total := w.w * w.h
	if total == 0 || len(w.rainCurr) != total || len(w.rainNext) != total {
		return
	}

	for i := range w.rainNext {
		w.rainNext[i] = 0
	}

	nextRegions := w.rainRegions[:0]
	for i := range w.rainRegions {
		region := w.rainRegions[i]
		if region.ttl <= 0 {
			continue
		}
		w.rasterizeRainRegion(region)
		region.ttl--
		if region.ttl > 0 {
			nextRegions = append(nextRegions, region)
		}
	}

	w.rainRegions = nextRegions
	w.rainCurr, w.rainNext = w.rainNext, w.rainCurr
}

func (w *World) rasterizeRainRegion(region rainRegion) {
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

	if minX > maxX || minY > maxY {
		return
	}

	sigma := region.radius * 0.5
	if sigma <= 0 {
		sigma = region.radius
	}
	invTwoSigmaSq := 0.0
	if sigma > 0 {
		invTwoSigmaSq = 1.0 / (2 * sigma * sigma)
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

			intensity := region.strength
			if invTwoSigmaSq > 0 {
				intensity *= math.Exp(-dist * dist * invTwoSigmaSq)
			}
			if intensity <= 0 {
				continue
			}
			if intensity > 1 {
				intensity = 1
			}
			if intensity < 0.001 {
				continue
			}

			idx := y*w.w + x
			if idx < 0 || idx >= len(w.rainNext) {
				continue
			}
			val := float32(intensity)
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
	if len(w.rainRegions) >= maxRegions {
		return
	}

	spawnChance := w.cfg.Params.RainSpawnChance
	if spawnChance <= 0 {
		return
	}
	if spawnChance > 1 {
		spawnChance = 1
	}
	if w.rng.Float64() >= spawnChance {
		return
	}

	radiusMin := w.cfg.Params.RainRadiusMin
	radiusMax := w.cfg.Params.RainRadiusMax
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

	ttlMin := w.cfg.Params.RainTTLMin
	ttlMax := w.cfg.Params.RainTTLMax
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

	strengthMin := w.cfg.Params.RainStrengthMin
	strengthMax := w.cfg.Params.RainStrengthMax
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

	cx := float64(w.rng.Intn(w.w)) + 0.5
	cy := float64(w.rng.Intn(w.h)) + 0.5

	w.rainRegions = append(w.rainRegions, rainRegion{
		cx:       cx,
		cy:       cy,
		radius:   float64(radius),
		strength: strength,
		ttl:      ttl,
	})
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
	if idx < len(w.display) {
		w.display[idx] = 1
	}
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
