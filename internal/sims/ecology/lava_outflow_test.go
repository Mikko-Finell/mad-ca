package ecology

import (
	"math"
	"testing"
)

func TestLavaOutflowExceedsVolcanoDiameter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 160
	cfg.Height = 160
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RockChance = 0
	cfg.Params.RainSpawnChance = 0
	cfg.Params.RainMaxRegions = 0
	cfg.Params.VolcanoProtoSpawnChance = 0

	cfg.Params.LavaReservoirMin = 700
	cfg.Params.LavaReservoirMax = 900
	cfg.Params.LavaReservoirGain = 2.4
	cfg.Params.LavaReservoirHead = 6.0
	cfg.Params.LavaCoolBase = 0.004
	cfg.Params.LavaCoolEdge = 0.0
	cfg.Params.LavaCoolThick = 0.006
	cfg.Params.LavaCoolFlux = 0.004
	cfg.Params.LavaCoolRain = 0.0
	cfg.Params.LavaSpreadChance = 0.30
	cfg.Params.LavaFluxRef = 4.2

	world := NewWithConfig(cfg)
	world.Reset(1337)

	ground := world.Ground()
	for i := range ground {
		ground[i] = GroundRock
	}
	veg := world.Vegetation()
	for i := range veg {
		veg[i] = VegetationNone
	}

	cx := cfg.Width / 2
	cy := cfg.Height / 2
	world.SpawnVolcanoAt(cx, cy)

	center := struct{ x, y float64 }{
		x: float64(cx) + 0.5,
		y: float64(cy) + 0.5,
	}

	maxDist := 0.0
	steps := 600
	for step := 0; step < steps; step++ {
		world.Step()
		ground := world.Ground()
		for idx, tile := range ground {
			if tile != GroundLava {
				continue
			}
			x := float64(idx%cfg.Width) + 0.5
			y := float64(idx/cfg.Width) + 0.5
			d := math.Hypot(x-center.x, y-center.y)
			if d > maxDist {
				maxDist = d
			}
		}
	}

	radius := float64(cfg.Params.VolcanoProtoRadiusMin)
	if cfg.Params.VolcanoProtoRadiusMax > cfg.Params.VolcanoProtoRadiusMin {
		radius = float64(cfg.Params.VolcanoProtoRadiusMin + (cfg.Params.VolcanoProtoRadiusMax-cfg.Params.VolcanoProtoRadiusMin)/2)
	}
	target := radius * 2
	if maxDist < target {
		t.Fatalf("lava max distance %.2f < target %.2f", maxDist, target)
	}
}
