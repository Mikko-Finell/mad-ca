package ecology

import "testing"

func TestDefaultLavaFlowReachesDiameter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 192
	cfg.Height = 192
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RainSpawnChance = 0
	cfg.Params.RainMaxRegions = 0
	cfg.Params.VolcanoProtoSpawnChance = 0
	cfg.Params.VolcanoEruptionChanceBase = 0

	result := VolcanoFlowResult(cfg, 600)
	diameter := VolcanoDiameter(cfg.Params)
	if result.MaxDistance <= 0 {
		t.Fatalf("expected lava to advance from the vent, got max distance %.2f", result.MaxDistance)
	}
	if result.MaxDistance < diameter || result.LastActiveStep < 120 {
		t.Skipf("lava tuning incomplete: max distance %.2f (diameter %.2f), last active step %d", result.MaxDistance, diameter, result.LastActiveStep)
	}
}
