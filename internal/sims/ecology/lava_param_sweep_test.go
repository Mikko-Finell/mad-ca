package ecology

import (
	"math"
	"testing"
)

type lavaSweepCandidate struct {
	reservoirMin    int
	reservoirMax    int
	gain            float64
	head            float64
	coolBase        float64
	coolEdge        float64
	coolThick       float64
	coolFlux        float64
	fluxRef         float64
	phaseThreshold  float64
	phaseHysteresis float64
	description     string
}

func (c lavaSweepCandidate) apply(cfg *Config) {
	cfg.Params.LavaReservoirMin = c.reservoirMin
	cfg.Params.LavaReservoirMax = c.reservoirMax
	cfg.Params.LavaReservoirGain = c.gain
	cfg.Params.LavaReservoirHead = c.head
	cfg.Params.LavaCoolBase = c.coolBase
	cfg.Params.LavaCoolEdge = c.coolEdge
	cfg.Params.LavaCoolThick = c.coolThick
	cfg.Params.LavaCoolFlux = c.coolFlux
	cfg.Params.LavaFluxRef = c.fluxRef
	if c.phaseThreshold > 0 {
		cfg.Params.LavaPhaseThreshold = c.phaseThreshold
	}
	if c.phaseHysteresis > 0 {
		cfg.Params.LavaPhaseHysteresis = c.phaseHysteresis
	}
}

type lavaTrialResult struct {
	maxDistance        float64
	ventCount          int
	tipCount           int
	centerElevation    int16
	tipElevation       int16
	tipNeighborLowest  int16
	tipNeighborHighest int16
	tipDistance        float64
}

func TestLavaParameterSweepFindsOutflow(t *testing.T) {
	baseCfg := DefaultConfig()
	baseCfg.Width = 96
	baseCfg.Height = 96
	baseCfg.Params.GrassPatchCount = 0
	baseCfg.Params.RainMaxRegions = 0
	baseCfg.Params.RainSpawnChance = 0
	baseCfg.Params.VolcanoProtoSpawnChance = 0

	radius := float64(baseCfg.Params.VolcanoProtoRadiusMin+baseCfg.Params.VolcanoProtoRadiusMax) / 2
	target := radius * 2

	if false {
		cfg := baseCfg
		cfg.Seed = 4096
		result := runLavaOutflowTrial(cfg, radius, 6000)
		t.Fatalf("debug maxDist=%.2f tipDist=%.2f", result.maxDistance, result.tipDistance)
	}

	defaults := lavaSweepCandidate{
		reservoirMin:    baseCfg.Params.LavaReservoirMin,
		reservoirMax:    baseCfg.Params.LavaReservoirMax,
		gain:            baseCfg.Params.LavaReservoirGain,
		head:            baseCfg.Params.LavaReservoirHead,
		coolBase:        baseCfg.Params.LavaCoolBase,
		coolEdge:        baseCfg.Params.LavaCoolEdge,
		coolThick:       baseCfg.Params.LavaCoolThick,
		coolFlux:        baseCfg.Params.LavaCoolFlux,
		fluxRef:         baseCfg.Params.LavaFluxRef,
		phaseThreshold:  baseCfg.Params.LavaPhaseThreshold,
		phaseHysteresis: baseCfg.Params.LavaPhaseHysteresis,
		description:     "default",
	}

	candidates := []lavaSweepCandidate{
		defaults,
		{description: "hotter head", reservoirMin: 200, reservoirMax: 360, gain: 1.1, head: 5.2, coolBase: 0.015, coolEdge: 0.02, coolThick: 0.015, coolFlux: 0.012, fluxRef: 3, phaseThreshold: baseCfg.Params.LavaPhaseThreshold, phaseHysteresis: baseCfg.Params.LavaPhaseHysteresis},
		{description: "reduced cooling", reservoirMin: 220, reservoirMax: 380, gain: 1.2, head: 4.8, coolBase: 0.008, coolEdge: 0.015, coolThick: 0.008, coolFlux: 0.008, fluxRef: 3.5, phaseThreshold: 0.12, phaseHysteresis: 0.04},
		{description: "low head spill", reservoirMin: 240, reservoirMax: 400, gain: 1.3, head: 2.2, coolBase: 0.006, coolEdge: 0.012, coolThick: 0.006, coolFlux: 0.006, fluxRef: 4, phaseThreshold: 0.1, phaseHysteresis: 0.05},
		{description: "very low head", reservoirMin: 260, reservoirMax: 420, gain: 1.4, head: 1.4, coolBase: 0.004, coolEdge: 0.01, coolThick: 0.004, coolFlux: 0.005, fluxRef: 4.5, phaseThreshold: 0.08, phaseHysteresis: 0.04},
		{description: "massive reservoir", reservoirMin: 320, reservoirMax: 520, gain: 1.5, head: 1.8, coolBase: 0.005, coolEdge: 0.012, coolThick: 0.005, coolFlux: 0.004, fluxRef: 5, phaseThreshold: 0.08, phaseHysteresis: 0.05},
		{description: "extreme flood", reservoirMin: 360, reservoirMax: 640, gain: 1.8, head: 6.0, coolBase: 0.002, coolEdge: 0.005, coolThick: 0.002, coolFlux: 0.002, fluxRef: 6, phaseThreshold: 0.06, phaseHysteresis: 0.03},
		{description: "endless flood", reservoirMin: 600, reservoirMax: 900, gain: 2.4, head: 6.5, coolBase: 0.001, coolEdge: 0.003, coolThick: 0.001, coolFlux: 0.001, fluxRef: 8, phaseThreshold: 0.05, phaseHysteresis: 0.03},
		{description: "cataclysmic flood", reservoirMin: 900, reservoirMax: 1400, gain: 3.0, head: 8.0, coolBase: 0.0005, coolEdge: 0.0025, coolThick: 0.0008, coolFlux: 0.0008, fluxRef: 10, phaseThreshold: 0.045, phaseHysteresis: 0.025},
	}

	seeds := []int64{1337, 42, 777}
	steps := 2000
	bestDist := 0.0
	best := defaults
	bestSeed := baseCfg.Seed
	success := false

search:
	for idx, cand := range candidates {
		for _, seed := range seeds {
			cfg := baseCfg
			cfg.Seed = seed
			cand.apply(&cfg)
			result := runLavaOutflowTrial(cfg, radius, steps)
			if result.maxDistance > bestDist {
				bestDist = result.maxDistance
				best = cand
				bestSeed = seed
			}
			t.Logf("candidate %d (%s) seed=%d: head=%.2f gain=%.2f reservoir=[%d,%d] coolBase=%.3f coolEdge=%.3f coolThick=%.3f coolFlux=%.3f fluxRef=%.2f phase=%.3f hyst=%.3f => maxDist=%.2f vents=%d tips=%d elev(center=%d tip=%d neigh[min=%d max=%d] tipDist=%.2f)", idx, cand.description, seed, cand.head, cand.gain, cand.reservoirMin, cand.reservoirMax, cand.coolBase, cand.coolEdge, cand.coolThick, cand.coolFlux, cand.fluxRef, cand.phaseThreshold, cand.phaseHysteresis, result.maxDistance, result.ventCount, result.tipCount, result.centerElevation, result.tipElevation, result.tipNeighborLowest, result.tipNeighborHighest, result.tipDistance)
			if result.maxDistance >= target {
				success = true
				bestDist = result.maxDistance
				best = cand
				bestSeed = seed
				break search
			}
		}
	}

	if !success {
		t.Fatalf("no candidate reached target distance %.2f (best %.2f with head=%.2f gain=%.2f reservoir=[%d,%d] coolBase=%.3f coolEdge=%.3f coolThick=%.3f coolFlux=%.3f fluxRef=%.2f phase=%.3f hyst=%.3f seed=%d)", target, bestDist, best.head, best.gain, best.reservoirMin, best.reservoirMax, best.coolBase, best.coolEdge, best.coolThick, best.coolFlux, best.fluxRef, best.phaseThreshold, best.phaseHysteresis, bestSeed)
	}
}

func runLavaOutflowTrial(cfg Config, radius float64, steps int) lavaTrialResult {
	world := NewWithConfig(cfg)
	world.Reset(0)

	centerX := float64(world.w)/2 + 0.5
	centerY := float64(world.h)/2 + 0.5

	region := volcanoProtoRegion{
		cx:       centerX,
		cy:       centerY,
		radius:   radius,
		strength: 1,
		ttl:      0,
	}
	world.eruptRegion(region)

	tipCount := 0
	tipElevation := int16(0)
	tipNeighborMin := int16(0)
	tipNeighborMax := int16(0)
	tipDist := 0.0
	for _, vent := range world.lavaVents {
		if vent.outIdx >= 0 && vent.outIdx < len(world.groundCurr) && vent.outIdx != vent.idx {
			tipCount++
			if tipElevation == 0 && vent.outIdx < len(world.lavaElevation) {
				tipElevation = world.lavaElevation[vent.outIdx]
				tipNeighborMin, tipNeighborMax = neighborElevationRange(world, vent.outIdx)
				tipDist = distanceToCell(world, vent.outIdx, region.cx, region.cy)
			}
		}
	}

	centerIdx := int(centerY)*world.w + int(centerX)
	if centerIdx < 0 || centerIdx >= len(world.lavaElevation) {
		centerIdx = 0
	}
	centerElevation := world.lavaElevation[centerIdx]

	maxDist := maxLavaDistance(world, region.cx, region.cy)
	for step := 0; step < steps; step++ {
		world.Step()
		dist := maxLavaDistance(world, region.cx, region.cy)
		if dist > maxDist {
			maxDist = dist
		}
	}
	return lavaTrialResult{maxDistance: maxDist, ventCount: len(world.lavaVents), tipCount: tipCount, centerElevation: centerElevation, tipElevation: tipElevation, tipNeighborLowest: tipNeighborMin, tipNeighborHighest: tipNeighborMax, tipDistance: tipDist}
}

func neighborElevationRange(world *World, idx int) (int16, int16) {
	minVal := int16(32767)
	maxVal := int16(-32768)
	x := idx % world.w
	y := idx / world.w
	for _, dir := range lavaDirections {
		nx := x + dir.dx
		ny := y + dir.dy
		if nx < 0 || nx >= world.w || ny < 0 || ny >= world.h {
			continue
		}
		nIdx := ny*world.w + nx
		elev := world.lavaElevation[nIdx]
		if elev < minVal {
			minVal = elev
		}
		if elev > maxVal {
			maxVal = elev
		}
	}
	if minVal == 32767 {
		minVal = 0
	}
	if maxVal == -32768 {
		maxVal = 0
	}
	return minVal, maxVal
}

func distanceToCell(world *World, idx int, cx, cy float64) float64 {
	x := float64(idx%world.w) + 0.5
	y := float64(idx/world.w) + 0.5
	dx := x - cx
	dy := y - cy
	return math.Sqrt(dx*dx + dy*dy)
}

func maxLavaDistance(world *World, cx, cy float64) float64 {
	maxDist := 0.0
	total := world.w * world.h
	for idx := 0; idx < total; idx++ {
		if world.groundCurr[idx] != GroundLava {
			continue
		}
		x := float64(idx%world.w) + 0.5
		y := float64(idx/world.w) + 0.5
		dx := x - cx
		dy := y - cy
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > maxDist {
			maxDist = dist
		}
	}
	return maxDist
}
