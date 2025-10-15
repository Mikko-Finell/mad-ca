package ecology

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
	"testing"
)

type lavaScenarioResult struct {
	MaxDistance     float64
	TicksSimulated  int
	ActiveLavaTicks int
}

func TestDefaultVolcanoLavaReach(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 112
	cfg.Height = 112
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RainSpawnChance = 0
	cfg.Params.VolcanoProtoSpawnChance = 0

	const volcanoRadius = 20
	const maxTicks = 1200
	targetDistance := 2 * float64(volcanoRadius)

	result := simulateCentralEruption(cfg, volcanoRadius, maxTicks, targetDistance)
	if result.MaxDistance >= targetDistance {
		t.Logf("default params reached max distance %.2f after %d ticks", result.MaxDistance, result.ActiveLavaTicks)
		return
	}

	bestParams, bestResult, ok := sweepLavaParameters(cfg, volcanoRadius, maxTicks, targetDistance)
	if !ok {
		t.Fatalf("lava failed to reach diameter: best %.2f < %.2f after sweep using params %+v", bestResult.MaxDistance, targetDistance, bestParams)
	}

	t.Fatalf("default params insufficient; sweep found max %.2f with params %+v", bestResult.MaxDistance, bestParams)
}

func TestDebugLavaCandidate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 112
	cfg.Height = 112
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RainSpawnChance = 0
	cfg.Params.VolcanoProtoSpawnChance = 0

	cfg.Params.LavaCoolBase = 0.0005
	cfg.Params.LavaCoolFlux = 0.0004
	cfg.Params.LavaCoolEdge = 0.0015
	cfg.Params.LavaCoolThick = 0.0015
	cfg.Params.LavaFluxRef = 12
	cfg.Params.LavaFlowThreshold = -0.05
	cfg.Params.LavaReservoirGain = 14.0
	cfg.Params.LavaReservoirHead = 25.0
	cfg.Params.LavaReservoirMin = 860
	cfg.Params.LavaReservoirMax = 1340
	cfg.Params.LavaSpreadChance = 0.78
	cfg.Params.LavaSpreadMaskFloor = 0.85
	cfg.Params.LavaPhaseThreshold = 0.03
	cfg.Params.LavaPhaseHysteresis = 0.01
	cfg.Params.LavaSlopeMultiplier = 150.0

	world := NewWithConfig(cfg)
	world.Reset(0)

	total := world.w * world.h
	centerX := float64(world.w) / 2
	centerY := float64(world.h) / 2
	radius := 20

	region := volcanoProtoRegion{cx: centerX, cy: centerY, radius: float64(radius), strength: 1, ttl: 0}
	world.eruptRegion(region)

	t.Logf("vents=%d", len(world.lavaVents))
	for _, vent := range world.lavaVents {
		idx := vent.idx
		if idx < 0 || idx >= len(world.lavaElevation) {
			continue
		}
		vx := idx % world.w
		vy := idx / world.w
		elevHere := world.lavaElevation[idx]
		drop := 0
		if vent.dir >= 0 {
			dir := lavaDirections[vent.dir]
			nx := vx + dir.dx
			ny := vy + dir.dy
			if nx >= 0 && nx < world.w && ny >= 0 && ny < world.h {
				nIdx := ny*world.w + nx
				if nIdx >= 0 && nIdx < len(world.lavaElevation) {
					drop = int(elevHere) - int(world.lavaElevation[nIdx])
				}
			}
		}
		var profile []int
		var groundProfile []Ground
		if vent.dir >= 0 {
			dir := lavaDirections[vent.dir]
			px := vx
			py := vy
			base := int(elevHere)
			for step := 0; step < 6; step++ {
				px += dir.dx
				py += dir.dy
				if px < 0 || px >= world.w || py < 0 || py >= world.h {
					break
				}
				nIdx := py*world.w + px
				if nIdx < 0 || nIdx >= len(world.lavaElevation) {
					break
				}
				profile = append(profile, base-int(world.lavaElevation[nIdx]))
				if nIdx >= 0 && nIdx < len(world.groundCurr) {
					groundProfile = append(groundProfile, world.groundCurr[nIdx])
				}
			}
		}
		t.Logf("vent idx=%d dir=%d mass=%.1f elev=%d drop=%d profile=%v groundProfile=%v", idx, vent.dir, vent.massRemaining, elevHere, drop, profile, groundProfile)
	}
	cx := int(centerX)
	cy := int(centerY)
	centerIdx := cy*world.w + cx
	if centerIdx >= 0 && centerIdx < len(world.lavaElevation) {
		elevCenter := world.lavaElevation[centerIdx]
		var drops []int
		for _, dir := range lavaDirections {
			nx := cx + dir.dx
			ny := cy + dir.dy
			if nx < 0 || nx >= world.w || ny < 0 || ny >= world.h {
				continue
			}
			nIdx := ny*world.w + nx
			diff := int(elevCenter)
			if nIdx >= 0 && nIdx < len(world.lavaElevation) {
				diff -= int(world.lavaElevation[nIdx])
			}
			drops = append(drops, diff)
		}
		t.Logf("center elevation=%d drops=%v", elevCenter, drops)
	}

	maxDistance := 0.0
	const steps = 1200
	for i := 0; i < steps; i++ {
		distance, active := lavaExtent(world, centerX, centerY)
		if distance > maxDistance {
			maxDistance = distance
		}
		if i%100 == 0 {
			t.Logf("tick=%d distance=%.2f", i, distance)
		}
		if active == 0 {
			break
		}
		if i < 10 {
			for vi, vent := range world.lavaVents {
				height := 0
				if vent.idx >= 0 && vent.idx < len(world.lavaHeight) {
					height = int(world.lavaHeight[vent.idx])
				}
				outHeight := 0
				if vent.outIdx >= 0 && vent.outIdx < len(world.lavaHeight) {
					outHeight = int(world.lavaHeight[vent.outIdx])
				}
				outForce := false
				if vent.outIdx >= 0 && vent.outIdx < len(world.lavaForce) {
					outForce = world.lavaForce[vent.outIdx]
				}
				t.Logf("step=%d vent=%d mass=%.1f height=%d outHeight=%d outForce=%t", i, vi, vent.massRemaining, height, outHeight, outForce)
			}
			t.Logf("advanced cells=%d", len(world.lavaAdvancedCells))
		}
		world.Step()
	}
	lavaCells := 0
	outerCells := 0
	coreRadius := float64(radius) * 0.35
	radiusThreshold := coreRadius + 0.1
	for idx := 0; idx < total; idx++ {
		if world.groundCurr[idx] != GroundLava {
			continue
		}
		lavaCells++
		x := float64(idx%world.w) + 0.5
		y := float64(idx/world.w) + 0.5
		dist := math.Hypot(x-centerX, y-centerY)
		if dist > radiusThreshold {
			outerCells++
		}
	}

	t.Skipf("debug distance %.2f after %d steps lavaCells=%d outerCells=%d", maxDistance, steps, lavaCells, outerCells)
}

func simulateCentralEruption(cfg Config, radius int, ticks int, stopDistance float64) lavaScenarioResult {
	world := NewWithConfig(cfg)
	world.Reset(0)

	centerX := float64(world.w) / 2
	centerY := float64(world.h) / 2

	region := volcanoProtoRegion{
		cx:       centerX,
		cy:       centerY,
		radius:   float64(radius),
		strength: 1,
		ttl:      0,
	}

	world.eruptRegion(region)

	maxDistance := 0.0
	activeTicks := 0
	lastGainStep := 0
	const stagnationLimit = 800

	for step := 0; step < ticks; step++ {
		distance, active := lavaExtent(world, centerX, centerY)
		if distance > maxDistance {
			maxDistance = distance
			lastGainStep = step
		}
		if stopDistance > 0 && maxDistance >= stopDistance {
			activeTicks = step
			return lavaScenarioResult{MaxDistance: maxDistance, TicksSimulated: step, ActiveLavaTicks: activeTicks}
		}
		if active == 0 {
			return lavaScenarioResult{MaxDistance: maxDistance, TicksSimulated: step, ActiveLavaTicks: activeTicks}
		}
		if step-lastGainStep > stagnationLimit {
			return lavaScenarioResult{MaxDistance: maxDistance, TicksSimulated: step, ActiveLavaTicks: activeTicks}
		}
		world.Step()
		activeTicks = step + 1
	}

	distance, active := lavaExtent(world, centerX, centerY)
	if distance > maxDistance {
		maxDistance = distance
	}
	if active > 0 {
		activeTicks = ticks
	}

	return lavaScenarioResult{MaxDistance: maxDistance, TicksSimulated: ticks, ActiveLavaTicks: activeTicks}
}

func lavaExtent(world *World, centerX, centerY float64) (float64, int) {
	maxDistance := 0.0
	active := 0
	total := world.w * world.h
	for idx := 0; idx < total; idx++ {
		if world.groundCurr[idx] != GroundLava {
			continue
		}
		if world.lavaHeight[idx] == 0 {
			continue
		}
		active++
		x := float64(idx%world.w) + 0.5
		y := float64(idx/world.w) + 0.5
		distance := math.Hypot(x-centerX, y-centerY)
		if distance > maxDistance {
			maxDistance = distance
		}
	}
	return maxDistance, active
}

func sweepLavaParameters(base Config, radius, ticks int, targetDistance float64) (Params, lavaScenarioResult, bool) {
	type candidate struct {
		params Params
		result lavaScenarioResult
	}

	evaluate := func(p Params) candidate {
		cfg := base
		cfg.Params = p
		res := simulateCentralEruption(cfg, radius, ticks, targetDistance)
		return candidate{params: p, result: res}
	}

	best := evaluate(base.Params)
	if best.result.MaxDistance >= targetDistance {
		return best.params, best.result, true
	}

	rng := rand.New(rand.NewSource(1337))
	iterations := 96

	sampleLogRange := func(min, max float64) float64 {
		if min <= 0 {
			min = 1e-6
		}
		if max <= min {
			return min
		}
		ratio := max / min
		return min * math.Pow(ratio, rng.Float64())
	}

	sampleLinearRange := func(min, max float64) float64 {
		if max <= min {
			return min
		}
		return min + rng.Float64()*(max-min)
	}
	sampleSymmetric := func(center, span float64) float64 {
		if span <= 0 {
			return center
		}
		return center + (rng.Float64()*2-1)*span
	}

	samples := make([]Params, iterations)
	for i := 0; i < iterations; i++ {
		params := base.Params
		params.LavaCoolBase = sampleLogRange(params.LavaCoolBase*0.01, params.LavaCoolBase)
		params.LavaCoolFlux = sampleLogRange(params.LavaCoolFlux*0.01, params.LavaCoolFlux)
		params.LavaCoolEdge = sampleLogRange(params.LavaCoolEdge*0.03, params.LavaCoolEdge)
		params.LavaCoolThick = sampleLogRange(params.LavaCoolThick*0.03, params.LavaCoolThick)
		params.LavaFluxRef = sampleLogRange(params.LavaFluxRef*0.8, params.LavaFluxRef*12)
		params.LavaFlowThreshold = sampleSymmetric(0.15, 0.5)
		if params.LavaFlowThreshold < -0.5 {
			params.LavaFlowThreshold = -0.5
		}
		params.LavaSlopeMultiplier = sampleLinearRange(1, 200)
		params.LavaReservoirGain = sampleLogRange(math.Max(0.1, params.LavaReservoirGain*0.5), params.LavaReservoirGain*15)
		params.LavaReservoirHead = sampleLinearRange(params.LavaReservoirHead*0.75, params.LavaReservoirHead+28)

		minMass := int(sampleLinearRange(float64(params.LavaReservoirMin), 1400))
		extraMass := int(sampleLinearRange(200, 620))
		params.LavaReservoirMin = minMass
		params.LavaReservoirMax = minMass + extraMass

		params.LavaSpreadChance = sampleLinearRange(params.LavaSpreadChance*0.6, 0.95)
		params.LavaSpreadMaskFloor = math.Min(1, sampleLinearRange(math.Max(0.2, params.LavaSpreadMaskFloor*0.8), params.LavaSpreadMaskFloor+0.5))
		params.LavaPhaseThreshold = sampleLinearRange(params.LavaPhaseThreshold*0.1, params.LavaPhaseThreshold)
		params.LavaPhaseHysteresis = sampleLinearRange(params.LavaPhaseHysteresis*0.1, params.LavaPhaseHysteresis)

		samples[i] = params
	}

	results := make([]lavaScenarioResult, iterations)
	parallelism := runtime.GOMAXPROCS(0)
	if parallelism < 1 {
		parallelism = 1
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < parallelism; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				cfg := base
				cfg.Params = samples[idx]
				results[idx] = simulateCentralEruption(cfg, radius, ticks, targetDistance)
			}
		}()
	}

	for i := range samples {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	bestMeeting := candidate{}
	meetingFound := false
	for i, res := range results {
		params := samples[i]
		if res.MaxDistance > best.result.MaxDistance {
			best = candidate{params: params, result: res}
		}
		if res.MaxDistance >= targetDistance {
			if !meetingFound || res.MaxDistance > bestMeeting.result.MaxDistance {
				bestMeeting = candidate{params: params, result: res}
				meetingFound = true
			}
		}
	}
	if meetingFound {
		return bestMeeting.params, bestMeeting.result, true
	}

	return best.params, best.result, best.result.MaxDistance >= targetDistance
}
