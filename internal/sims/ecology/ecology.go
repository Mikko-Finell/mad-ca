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
}

// Step advances the simulation by applying the vegetation succession rules once.
func (w *World) Step() {
	if w.w == 0 || w.h == 0 {
		return
	}

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
