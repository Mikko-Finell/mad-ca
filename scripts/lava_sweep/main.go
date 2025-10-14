package main

import (
	"fmt"

	ecology "mad-ca/internal/sims/ecology"
)

type sweepParams struct {
	lavaSpreadChance    float64
	lavaSpreadMaskFloor float64
	lavaFluxRef         float64
	lavaCoolBase        float64
	lavaCoolEdge        float64
	lavaCoolThick       float64
	lavaCoolFlux        float64
	lavaPhaseThreshold  float64
	lavaPhaseHysteresis float64
	lavaSlopeScale      float64
	reservoirMin        int
	reservoirMax        int
	reservoirGain       float64
	reservoirHead       float64
}

func main() {
	candidates := []sweepParams{
		{
			lavaSpreadChance:    0.75,
			lavaSpreadMaskFloor: 0.02,
			lavaFluxRef:         9.0,
			lavaCoolBase:        0.002,
			lavaCoolEdge:        0.003,
			lavaCoolThick:       0.002,
			lavaCoolFlux:        0.0015,
			lavaPhaseThreshold:  0.06,
			lavaPhaseHysteresis: 0.02,
			lavaSlopeScale:      3,
			reservoirMin:        520,
			reservoirMax:        760,
			reservoirGain:       1.9,
			reservoirHead:       8.0,
		},
		{
			lavaSpreadChance:    0.85,
			lavaSpreadMaskFloor: 0.01,
			lavaFluxRef:         9.5,
			lavaCoolBase:        0.0015,
			lavaCoolEdge:        0.0025,
			lavaCoolThick:       0.0015,
			lavaCoolFlux:        0.001,
			lavaPhaseThreshold:  0.055,
			lavaPhaseHysteresis: 0.018,
			lavaSlopeScale:      7,
			reservoirMin:        540,
			reservoirMax:        800,
			reservoirGain:       2.0,
			reservoirHead:       8.4,
		},
		{
			lavaSpreadChance:    1.0,
			lavaSpreadMaskFloor: 0.0,
			lavaFluxRef:         10.0,
			lavaCoolBase:        0.0012,
			lavaCoolEdge:        0.002,
			lavaCoolThick:       0.0012,
			lavaCoolFlux:        0.0009,
			lavaPhaseThreshold:  0.05,
			lavaPhaseHysteresis: 0.018,
			lavaSlopeScale:      6,
			reservoirMin:        560,
			reservoirMax:        820,
			reservoirGain:       2.1,
			reservoirHead:       8.6,
		},
	}

	fmt.Printf("evaluating %d parameter combinations\n", len(candidates))
	for _, params := range candidates {
		reach, diameter := simulate(params)
		fmt.Printf("reach %.2f vs diameter %.2f with params: maskFloor=%.2f fluxRef=%.2f base=%.4f edge=%.4f thick=%.4f flux=%.4f phase=%.3f hyst=%.3f min=%d max=%d gain=%.2f head=%.2f\n",
			reach, diameter, params.lavaSpreadMaskFloor, params.lavaFluxRef, params.lavaCoolBase,
			params.lavaCoolEdge, params.lavaCoolThick, params.lavaCoolFlux, params.lavaPhaseThreshold,
			params.lavaPhaseHysteresis, params.reservoirMin, params.reservoirMax, params.reservoirGain, params.reservoirHead)
	}
}

func simulate(params sweepParams) (float64, float64) {
	cfg := ecology.DefaultConfig()
	cfg.Width = 128
	cfg.Height = 128
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RainSpawnChance = 0
	cfg.Params.RainMaxRegions = 0
	cfg.Params.VolcanoProtoSpawnChance = 0
	cfg.Params.VolcanoProtoMaxRegions = 1
	cfg.Params.VolcanoProtoRadiusMin = 2
	cfg.Params.VolcanoProtoRadiusMax = 4

	cfg.Params.LavaSpreadChance = params.lavaSpreadChance
	cfg.Params.LavaSpreadMaskFloor = params.lavaSpreadMaskFloor
	cfg.Params.LavaFluxRef = params.lavaFluxRef
	cfg.Params.LavaCoolBase = params.lavaCoolBase
	cfg.Params.LavaCoolEdge = params.lavaCoolEdge
	cfg.Params.LavaCoolThick = params.lavaCoolThick
	cfg.Params.LavaCoolFlux = params.lavaCoolFlux
	cfg.Params.LavaPhaseThreshold = params.lavaPhaseThreshold
	cfg.Params.LavaPhaseHysteresis = params.lavaPhaseHysteresis
	cfg.Params.LavaSlopeScale = params.lavaSlopeScale
	cfg.Params.LavaReservoirMin = params.reservoirMin
	cfg.Params.LavaReservoirMax = params.reservoirMax
	cfg.Params.LavaReservoirGain = params.reservoirGain
	cfg.Params.LavaReservoirHead = params.reservoirHead

	world := ecology.NewWithConfig(cfg)
	world.Reset(0)

	cx := cfg.Width / 2
	cy := cfg.Height / 2
	world.SpawnVolcanoAt(cx, cy)
	fmt.Printf("vent summary: %+v\n", world.LavaVentStatuses())

	radius := float64(cfg.Params.VolcanoProtoRadiusMin)
	if cfg.Params.VolcanoProtoRadiusMax > cfg.Params.VolcanoProtoRadiusMin {
		radius = float64(cfg.Params.VolcanoProtoRadiusMin + (cfg.Params.VolcanoProtoRadiusMax-cfg.Params.VolcanoProtoRadiusMin)/2)
	}
	diameter := radius * 2

	centerX := float64(cx) + 0.5
	centerY := float64(cy) + 0.5

	reach := world.FarthestLavaDistanceFrom(centerX, centerY)
	fmt.Printf("initial lava tiles: %d\n", world.LavaTileCount())

	ticks := 600
	for i := 0; i < ticks; i++ {
		world.Step()
		if i == 0 {
			fmt.Printf("after first step lava tiles: %d, tips: %d, advances: %d, max height: %d\n", world.LavaTileCount(), world.LavaTipCount(), world.LavaAdvanceCount(), world.MaxLavaHeight())
		}
		if i == 50 {
			fmt.Printf("after 50 steps lava tiles: %d, tips: %d, advances: %d, max height: %d\n", world.LavaTileCount(), world.LavaTipCount(), world.LavaAdvanceCount(), world.MaxLavaHeight())
		}
		if r := world.FarthestLavaDistanceFrom(centerX, centerY); r > reach {
			reach = r
		}
		if world.LavaTileCount() == 0 {
			break
		}
	}

	return reach, diameter
}
