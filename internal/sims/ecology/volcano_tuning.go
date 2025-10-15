package ecology

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
)

// LavaFlowResult captures telemetry from a deterministic volcano run used for tuning.
type LavaFlowResult struct {
	// MaxDistance records the farthest Euclidean distance (in tiles) that
	// lava reached from the volcano centre.
	MaxDistance float64
	// MaxDistanceStep stores the tick at which the farthest distance was first achieved.
	MaxDistanceStep int
	// PeakActiveTiles tracks the maximum number of lava tiles present at any step.
	PeakActiveTiles int
	// LastActiveStep records the final tick that still contained active lava.
	LastActiveStep int
	// StepsSimulated reports how many ticks the simulation executed (including the initial state).
	StepsSimulated int
	// InitialTipCount captures how many lava tips were seeded by the eruption.
	InitialTipCount int
}

// SweepRecord documents a single improvement encountered while exploring the
// tuning parameter space.
type SweepRecord struct {
	Pass      int
	Parameter string
	Value     string
	Result    LavaFlowResult
	Params    Params
}

// VolcanoDiameter returns the nominal diameter implied by the proto-volcano
// radius tunables.
func VolcanoDiameter(p Params) float64 {
	radiusMin := p.VolcanoProtoRadiusMin
	if radiusMin <= 0 {
		radiusMin = 1
	}
	radiusMax := p.VolcanoProtoRadiusMax
	if radiusMax < radiusMin {
		radiusMax = radiusMin
	}
	radius := float64(midpointInt(radiusMin, radiusMax))
	return radius * 2
}

// VolcanoFlowResult runs a deterministic volcano scenario with the provided
// configuration and returns the lava spread telemetry.
//
// The helper resets the world to bare rock, spawns a volcano at the map
// centre, advances the simulation for the requested number of steps, and keeps
// the farthest lava distance along with activity duration statistics.
func VolcanoFlowResult(cfg Config, steps int) LavaFlowResult {
	if steps <= 0 {
		return LavaFlowResult{}
	}

	world := NewWithConfig(cfg)
	world.Reset(0)

	total := len(world.groundCurr)
	for i := 0; i < total; i++ {
		world.groundCurr[i] = GroundRock
		world.groundNext[i] = GroundRock
		world.display[i] = uint8(GroundRock)
	}

	cx := cfg.Width / 2
	cy := cfg.Height / 2
	world.SpawnVolcanoAt(cx, cy)

	centreX := float64(cx) + 0.5
	centreY := float64(cy) + 0.5

	result := LavaFlowResult{}

	for _, tip := range world.lavaTip {
		if tip {
			result.InitialTipCount++
		}
	}

	measureStep := func(step int) int {
		maxDistance := result.MaxDistance
		peakTiles := result.PeakActiveTiles
		lavaTiles := 0

		width := world.w
		for idx, ground := range world.groundCurr {
			if ground != GroundLava {
				continue
			}
			x := idx % width
			y := idx / width
			dx := float64(x) + 0.5 - centreX
			dy := float64(y) + 0.5 - centreY
			dist := math.Hypot(dx, dy)
			if dist > maxDistance {
				maxDistance = dist
				result.MaxDistanceStep = step
			}
			lavaTiles++
		}

		if maxDistance > result.MaxDistance {
			result.MaxDistance = maxDistance
		}
		if lavaTiles > peakTiles {
			peakTiles = lavaTiles
		}
		result.PeakActiveTiles = peakTiles
		if lavaTiles > 0 {
			result.LastActiveStep = step
		}
		return lavaTiles
	}

	inactiveLimit := 32
	inactive := 0

	tiles := measureStep(0)
	if tiles == 0 {
		inactive++
	}

	for step := 1; step <= steps; step++ {
		world.Step()
		tiles = measureStep(step)
		result.StepsSimulated = step
		if tiles == 0 {
			inactive++
			if inactive >= inactiveLimit {
				break
			}
			continue
		}
		inactive = 0
	}

	if result.StepsSimulated == 0 {
		result.StepsSimulated = steps
	}

	return result
}

type floatSpec struct {
	name   string
	values []float64
	getter func(Params) float64
	setter func(*Params, float64)
}

type intSpec struct {
	name   string
	values []int
	getter func(Params) int
	setter func(*Params, int)
	skip   func(Params, int) bool
}

// VolcanoParameterSweep performs a coarse coordinate-descent search across
// relevant lava parameters and returns the best parameter set discovered along
// with the associated telemetry and an improvement trace.
func VolcanoParameterSweep(base Config, steps, passes, workers int) (Params, LavaFlowResult, []SweepRecord) {
	if steps <= 0 {
		steps = 400
	}
	if passes <= 0 {
		passes = 1
	}
	if workers <= 0 {
		workers = 1
	}

	currentParams := base.Params
	currentResult := VolcanoFlowResult(applyParams(base, currentParams), steps)

	records := []SweepRecord{{
		Pass:      0,
		Parameter: "baseline",
		Value:     "",
		Result:    currentResult,
		Params:    currentParams,
	}}

	randomSamples := passes * 8
	if randomSamples < 16 {
		randomSamples = 16
	}
	rng := rand.New(rand.NewSource(base.Seed + 0x5f3759df))
	for i := 0; i < randomSamples; i++ {
		candidate := randomizeParams(rng, base.Params)
		cfg := applyParams(base, candidate)
		res := VolcanoFlowResult(cfg, steps)
		if betterLavaResult(res, currentResult) {
			currentParams = candidate
			currentResult = res
			records = append(records, SweepRecord{
				Pass:      0,
				Parameter: fmt.Sprintf("random#%d", i+1),
				Value:     "",
				Result:    res,
				Params:    candidate,
			})
		}
	}

	floatSpecs := []floatSpec{
		{
			name:   "lava_flux_ref",
			values: []float64{1.5, 2.0, 2.5, 3.0, 3.5},
			getter: func(p Params) float64 { return p.LavaFluxRef },
			setter: func(p *Params, v float64) { p.LavaFluxRef = v },
		},
		{
			name:   "lava_cool_base",
			values: []float64{0.005, 0.01, 0.015, 0.02, 0.025},
			getter: func(p Params) float64 { return p.LavaCoolBase },
			setter: func(p *Params, v float64) { p.LavaCoolBase = v },
		},
		{
			name:   "lava_cool_edge",
			values: []float64{0.015, 0.02, 0.025, 0.03},
			getter: func(p Params) float64 { return p.LavaCoolEdge },
			setter: func(p *Params, v float64) { p.LavaCoolEdge = v },
		},
		{
			name:   "lava_cool_thick",
			values: []float64{0.01, 0.015, 0.02, 0.025},
			getter: func(p Params) float64 { return p.LavaCoolThick },
			setter: func(p *Params, v float64) { p.LavaCoolThick = v },
		},
		{
			name:   "lava_cool_flux",
			values: []float64{0.01, 0.015, 0.02, 0.025},
			getter: func(p Params) float64 { return p.LavaCoolFlux },
			setter: func(p *Params, v float64) { p.LavaCoolFlux = v },
		},
		{
			name:   "lava_reservoir_gain",
			values: []float64{0.6, 0.8, 1.0, 1.2, 1.4},
			getter: func(p Params) float64 { return p.LavaReservoirGain },
			setter: func(p *Params, v float64) { p.LavaReservoirGain = v },
		},
		{
			name:   "lava_reservoir_head",
			values: []float64{3.5, 4.0, 4.5, 5.0, 5.5, 6.0},
			getter: func(p Params) float64 { return p.LavaReservoirHead },
			setter: func(p *Params, v float64) { p.LavaReservoirHead = v },
		},
	}

	intSpecs := []intSpec{
		{
			name:   "lava_reservoir_min",
			values: []int{120, 160, 200, 240, 280},
			getter: func(p Params) int { return p.LavaReservoirMin },
			setter: func(p *Params, v int) {
				if v < 0 {
					v = 0
				}
				p.LavaReservoirMin = v
				if p.LavaReservoirMax < v {
					p.LavaReservoirMax = v
				}
			},
		},
		{
			name:   "lava_reservoir_max",
			values: []int{220, 260, 320, 380, 440},
			getter: func(p Params) int { return p.LavaReservoirMax },
			setter: func(p *Params, v int) {
				if v < p.LavaReservoirMin {
					v = p.LavaReservoirMin
				}
				p.LavaReservoirMax = v
			},
			skip: func(p Params, v int) bool { return v < p.LavaReservoirMin },
		},
	}

	for pass := 1; pass <= passes; pass++ {
		improved := false

		for _, spec := range intSpecs {
			bestParams, bestResult, changed, rec := evaluateIntSpec(base, currentParams, currentResult, spec, steps, workers, pass)
			if changed {
				currentParams = bestParams
				currentResult = bestResult
				records = append(records, rec...)
				improved = true
			}
		}

		for _, spec := range floatSpecs {
			bestParams, bestResult, changed, rec := evaluateFloatSpec(base, currentParams, currentResult, spec, steps, workers, pass)
			if changed {
				currentParams = bestParams
				currentResult = bestResult
				records = append(records, rec...)
				improved = true
			}
		}

		if !improved {
			break
		}
	}

	return currentParams, currentResult, records
}

func evaluateIntSpec(base Config, params Params, baseline LavaFlowResult, spec intSpec, steps, workers, pass int) (Params, LavaFlowResult, bool, []SweepRecord) {
	bestParams := params
	bestResult := baseline
	changed := false
	records := make([]SweepRecord, 0)

	type candidate struct {
		value  int
		result LavaFlowResult
		valid  bool
	}

	candidates := make([]candidate, len(spec.values))
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for idx, value := range spec.values {
		if value == spec.getter(params) {
			continue
		}
		if spec.skip != nil && spec.skip(params, value) {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i, v int) {
			defer wg.Done()
			candidateParams := params
			spec.setter(&candidateParams, v)
			cfg := applyParams(base, candidateParams)
			res := VolcanoFlowResult(cfg, steps)
			candidates[i] = candidate{value: v, result: res, valid: true}
			<-sem
		}(idx, value)
	}

	wg.Wait()

	for idx, value := range spec.values {
		cand := candidates[idx]
		if !cand.valid {
			continue
		}
		if betterLavaResult(cand.result, bestResult) {
			candidateParams := params
			spec.setter(&candidateParams, value)
			bestParams = candidateParams
			bestResult = cand.result
			changed = true
			records = append(records, SweepRecord{
				Pass:      pass,
				Parameter: spec.name,
				Value:     strconv.Itoa(value),
				Result:    cand.result,
				Params:    candidateParams,
			})
		}
	}

	return bestParams, bestResult, changed, records
}

func evaluateFloatSpec(base Config, params Params, baseline LavaFlowResult, spec floatSpec, steps, workers, pass int) (Params, LavaFlowResult, bool, []SweepRecord) {
	bestParams := params
	bestResult := baseline
	changed := false
	records := make([]SweepRecord, 0)

	type candidate struct {
		value  float64
		result LavaFlowResult
		valid  bool
	}

	candidates := make([]candidate, len(spec.values))
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for idx, value := range spec.values {
		if almostEqual(value, spec.getter(params)) {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i int, v float64) {
			defer wg.Done()
			candidateParams := params
			spec.setter(&candidateParams, v)
			cfg := applyParams(base, candidateParams)
			res := VolcanoFlowResult(cfg, steps)
			candidates[i] = candidate{value: v, result: res, valid: true}
			<-sem
		}(idx, value)
	}

	wg.Wait()

	for idx, value := range spec.values {
		cand := candidates[idx]
		if !cand.valid {
			continue
		}
		if betterLavaResult(cand.result, bestResult) {
			candidateParams := params
			spec.setter(&candidateParams, value)
			bestParams = candidateParams
			bestResult = cand.result
			changed = true
			records = append(records, SweepRecord{
				Pass:      pass,
				Parameter: spec.name,
				Value:     fmt.Sprintf("%.3f", value),
				Result:    cand.result,
				Params:    candidateParams,
			})
		}
	}

	return bestParams, bestResult, changed, records
}

func betterLavaResult(a, b LavaFlowResult) bool {
	if a.MaxDistance > b.MaxDistance {
		return true
	}
	if a.MaxDistance < b.MaxDistance {
		return false
	}
	return a.LastActiveStep > b.LastActiveStep
}

func almostEqual(a, b float64) bool {
	const eps = 1e-6
	return math.Abs(a-b) <= eps
}

func applyParams(base Config, params Params) Config {
	cfg := base
	cfg.Params = params
	return cfg
}

func randomizeParams(rng *rand.Rand, base Params) Params {
	params := base
	params.LavaFluxRef = randomFloatRange(rng, 0.5, 3.5)
	params.LavaCoolBase = randomFloatRange(rng, 0.0, 0.03)
	params.LavaCoolEdge = randomFloatRange(rng, 0.0, 0.04)
	params.LavaCoolThick = randomFloatRange(rng, 0.0, 0.03)
	params.LavaCoolFlux = randomFloatRange(rng, 0.0, 0.03)
	params.LavaReservoirGain = randomFloatRange(rng, 0.5, 2.0)
	params.LavaReservoirHead = randomFloatRange(rng, 3.5, 8.0)
	params.LavaPhaseThreshold = randomFloatRange(rng, 0.05, 0.2)
	params.LavaPhaseHysteresis = randomFloatRange(rng, 0.01, 0.08)
	min := randomIntRange(rng, 120, 480)
	params.LavaReservoirMin = min
	maxMin := min + 120
	max := randomIntRange(rng, maxMin, 800)
	if max < maxMin {
		max = maxMin
	}
	params.LavaReservoirMax = max
	return params
}

func randomFloatRange(rng *rand.Rand, min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + rng.Float64()*(max-min)
}

func randomIntRange(rng *rand.Rand, min, max int) int {
	if max <= min {
		return min
	}
	return rng.Intn(max-min+1) + min
}
