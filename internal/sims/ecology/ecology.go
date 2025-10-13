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
	rainCurr   []float32
	rainNext   []float32
	volCurr    []float32
	volNext    []float32
	tectonic   []float32
	display    []uint8

	rng *rand.Rand
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
}

// Step advances the simulation by applying the vegetation succession rules once.
func (w *World) Step() {
	if w.w == 0 || w.h == 0 {
		return
	}

	grassNeighbors, shrubNeighbors := w.mooreNeighborCounts()

	thresholdGrass := uint8(w.cfg.Params.GrassNeighborThreshold)
	thresholdShrub := uint8(w.cfg.Params.ShrubNeighborThreshold)
	thresholdTree := uint8(w.cfg.Params.TreeNeighborThreshold)

	total := w.w * w.h
	for i := 0; i < total; i++ {
		current := w.vegCurr[i]
		next := current

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

	w.vegCurr, w.vegNext = w.vegNext, w.vegCurr
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
