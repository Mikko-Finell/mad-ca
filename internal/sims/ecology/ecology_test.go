package ecology

import (
	"image/color"
	"math"
	"slices"
	"testing"
)

func TestResetDeterministic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 32
	cfg.Height = 24
	cfg.Seed = 99
	cfg.Params.RockChance = 0.2
	cfg.Params.GrassPatchCount = 6

	world := NewWithConfig(cfg)
	world.Reset(0)

	initialGround := append([]Ground(nil), world.Ground()...)
	initialVegetation := append([]Vegetation(nil), world.Vegetation()...)
	initialRain := append([]float32(nil), world.RainMask()...)
	initialVolcano := append([]float32(nil), world.VolcanoMask()...)
	initialCells := append([]uint8(nil), world.Cells()...)

	if len(initialGround) == 0 {
		t.Fatal("world must allocate ground layer")
	}

	// Mutate state to ensure Reset rebuilds from scratch.
	ground := world.Ground()
	ground[0] = GroundLava
	veg := world.Vegetation()
	veg[1] = VegetationTree
	world.RainMask()[2] = 1
	world.VolcanoMask()[3] = 1
	world.Cells()[4] = 42

	world.Reset(0)

	if !slices.Equal(initialGround, world.Ground()) {
		t.Fatal("Reset with config seed not deterministic for ground layer")
	}
	if !slices.Equal(initialVegetation, world.Vegetation()) {
		t.Fatal("Reset with config seed not deterministic for vegetation layer")
	}
	if !slices.Equal(initialRain, world.RainMask()) {
		t.Fatal("Reset with config seed not deterministic for rain mask")
	}
	if !slices.Equal(initialVolcano, world.VolcanoMask()) {
		t.Fatal("Reset with config seed not deterministic for volcano mask")
	}
	if !slices.Equal(initialCells, world.Cells()) {
		t.Fatal("Reset with config seed not deterministic for display buffer")
	}

	// Validate determinism for explicit seeds too.
	world.Reset(777)
	seedGround := append([]Ground(nil), world.Ground()...)
	seedVegetation := append([]Vegetation(nil), world.Vegetation()...)
	seedRain := append([]float32(nil), world.RainMask()...)
	seedVolcano := append([]float32(nil), world.VolcanoMask()...)
	seedCells := append([]uint8(nil), world.Cells()...)

	world.Reset(777)

	if !slices.Equal(seedGround, world.Ground()) {
		t.Fatal("Reset with explicit seed not deterministic for ground layer")
	}
	if !slices.Equal(seedVegetation, world.Vegetation()) {
		t.Fatal("Reset with explicit seed not deterministic for vegetation layer")
	}
	if !slices.Equal(seedRain, world.RainMask()) {
		t.Fatal("Reset with explicit seed not deterministic for rain mask")
	}
	if !slices.Equal(seedVolcano, world.VolcanoMask()) {
		t.Fatal("Reset with explicit seed not deterministic for volcano mask")
	}
	if !slices.Equal(seedCells, world.Cells()) {
		t.Fatal("Reset with explicit seed not deterministic for display buffer")
	}

	if slices.Equal(initialGround, seedGround) {
		t.Fatal("different seeds should produce different initial ground states")
	}
}

func TestSetFloatParameterVolcanoEruptionChance(t *testing.T) {
	cfg := DefaultConfig()
	world := NewWithConfig(cfg)

	if !world.SetFloatParameter("volcano_eruption_chance_base", 50) {
		t.Fatal("expected volcano eruption chance to be adjustable")
	}
	if got := world.cfg.Params.VolcanoEruptionChanceBase; math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("expected eruption chance 0.5, got %f", got)
	}

	if !world.SetFloatParameter("volcano_eruption_chance_base", 150) {
		t.Fatal("expected setter to clamp values above max")
	}
	if got := world.cfg.Params.VolcanoEruptionChanceBase; math.Abs(got-1) > 1e-9 {
		t.Fatalf("expected eruption chance to clamp to 1, got %f", got)
	}
}

func TestUpdateVolcanoMaskRasterizesAndExpires(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 8
	cfg.Height = 8
	cfg.Params.VolcanoProtoSpawnChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.volcanoRegions = []volcanoProtoRegion{{
		cx:       3.5,
		cy:       3.5,
		radius:   3,
		strength: 1,
		ttl:      2,
	}}

	world.updateVolcanoMask()

	if got := world.volCurr[3*world.w+3]; got <= 0 {
		t.Fatalf("expected mask center to be positive, got %f", got)
	}

	if len(world.volcanoRegions) != 1 {
		t.Fatalf("expected region to remain active with ttl>0, got %d", len(world.volcanoRegions))
	}

	world.updateVolcanoMask()

	if len(world.volcanoRegions) != 0 {
		t.Fatalf("expected region to expire, got %d active", len(world.volcanoRegions))
	}

	if len(world.expiredVolcanoProtos) == 0 {
		t.Fatal("expected expired region to be tracked")
	}

	world.updateVolcanoMask()

	if got := world.volCurr[3*world.w+3]; got != 0 {
		t.Fatalf("expected mask to clear after expiration tick, got %f", got)
	}
}

func TestApplyUpliftConvertsRockUnderMask(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 1
	cfg.Params.VolcanoUpliftChanceBase = 1
	cfg.Params.VolcanoProtoSpawnChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	for i := range world.groundCurr {
		world.groundCurr[i] = GroundRock
	}
	copy(world.groundNext, world.groundCurr)

	for i := range world.volCurr {
		world.volCurr[i] = 1
	}

	world.applyUplift()

	for i, v := range world.groundCurr {
		if v != GroundMountain {
			t.Fatalf("expected uplift to convert tile %d to mountain, got %v", i, v)
		}
	}
}

func TestSpawnVolcanoProtoRespectsTectonicThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 1
	cfg.Height = 1
	cfg.Params.VolcanoProtoSpawnChance = 1
	cfg.Params.VolcanoProtoMaxRegions = 2
	cfg.Params.VolcanoProtoTectonicThreshold = 0.9
	cfg.Params.VolcanoProtoRadiusMin = 5
	cfg.Params.VolcanoProtoRadiusMax = 5
	cfg.Params.VolcanoProtoTTLMin = 3
	cfg.Params.VolcanoProtoTTLMax = 3
	cfg.Params.VolcanoProtoStrengthMin = 1
	cfg.Params.VolcanoProtoStrengthMax = 1

	world := NewWithConfig(cfg)
	world.Reset(0)
	world.rng.Seed(12345)

	for i := range world.tectonic {
		world.tectonic[i] = 0
	}

	world.spawnVolcanoProtoRegion()

	if len(world.volcanoRegions) != 0 {
		t.Fatalf("expected no spawn when tectonic below threshold, got %d", len(world.volcanoRegions))
	}

	world.tectonic[0] = 1
	world.spawnVolcanoProtoRegion()

	if len(world.volcanoRegions) != 1 {
		t.Fatalf("expected spawn after threshold satisfied, got %d", len(world.volcanoRegions))
	}

	region := world.volcanoRegions[0]
	if region.ttl != cfg.Params.VolcanoProtoTTLMin {
		t.Fatalf("expected ttl %d, got %d", cfg.Params.VolcanoProtoTTLMin, region.ttl)
	}
	if math.Abs(region.radius-float64(cfg.Params.VolcanoProtoRadiusMin)) > 1e-6 {
		t.Fatalf("expected radius %d, got %f", cfg.Params.VolcanoProtoRadiusMin, region.radius)
	}
	if region.strength != cfg.Params.VolcanoProtoStrengthMin {
		t.Fatalf("expected strength %f, got %f", cfg.Params.VolcanoProtoStrengthMin, region.strength)
	}
}

func TestSpawnVolcanoAtEruptsImmediately(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 16
	cfg.Height = 12
	cfg.Params.VolcanoProtoRadiusMin = 4
	cfg.Params.VolcanoProtoRadiusMax = 10
	cfg.Params.VolcanoProtoTTLMin = 5
	cfg.Params.VolcanoProtoTTLMax = 11
	cfg.Params.VolcanoProtoStrengthMin = 0.25
	cfg.Params.VolcanoProtoStrengthMax = 0.75

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.SpawnVolcanoAt(6, 7)

	if len(world.volcanoRegions) != 0 {
		t.Fatalf("expected manual spawn to erupt immediately without proto regions, got %d", len(world.volcanoRegions))
	}

	if len(world.lavaVents) == 0 {
		t.Fatalf("expected eruption to create lava vents")
	}

	var lavaTiles int
	for _, g := range world.groundCurr {
		if g == GroundLava {
			lavaTiles++
		}
	}
	if lavaTiles == 0 {
		t.Fatalf("expected eruption to seed lava tiles")
	}
}

func TestSpawnVolcanoAtOutOfBoundsIgnored(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 8
	cfg.Height = 8

	world := NewWithConfig(cfg)
	world.Reset(0)

	initialLava := 0
	for _, g := range world.groundCurr {
		if g == GroundLava {
			initialLava++
		}
	}

	world.SpawnVolcanoAt(-1, 0)
	world.SpawnVolcanoAt(0, -1)
	world.SpawnVolcanoAt(8, 0)
	world.SpawnVolcanoAt(0, 8)

	if len(world.volcanoRegions) != 0 {
		t.Fatalf("expected out-of-bounds spawn attempts to be ignored, got %d regions", len(world.volcanoRegions))
	}

	if len(world.lavaVents) != 0 {
		t.Fatalf("expected out-of-bounds attempts to create no vents, got %d", len(world.lavaVents))
	}

	var lavaTiles int
	for _, g := range world.groundCurr {
		if g == GroundLava {
			lavaTiles++
		}
	}
	if lavaTiles != initialLava {
		t.Fatalf("expected lava tiles unchanged after ignored spawns, got %d want %d", lavaTiles, initialLava)
	}
}

func TestFireBurnsOutAndClears(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 1
	cfg.Seed = 1
	cfg.Params.GrassPatchCount = 0
	cfg.Params.FireSpreadChance = 0
	cfg.Params.BurnTTL = 2

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.vegCurr[1] = VegetationTree
	copy(world.vegNext, world.vegCurr)

	world.IgniteAt(1, 0)
	if got := int(world.burnTTL[1]); got != cfg.Params.BurnTTL {
		t.Fatalf("expected ignition to set burn ttl to %d, got %d", cfg.Params.BurnTTL, got)
	}

	world.Step()
	if got := int(world.burnTTL[1]); got != cfg.Params.BurnTTL-1 {
		t.Fatalf("expected burn ttl to decrement, got %d", got)
	}
	if world.vegCurr[1] != VegetationTree {
		t.Fatalf("vegetation should remain until burn completes, got %v", world.vegCurr[1])
	}

	world.Step()
	if world.burnTTL[1] != 0 {
		t.Fatalf("expected burn to expire, ttl=%d", world.burnTTL[1])
	}
	if world.vegCurr[1] != VegetationNone {
		t.Fatalf("expected vegetation to clear after burn, got %v", world.vegCurr[1])
	}
}

func TestFireSpreadsToNeighbor(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 1
	cfg.Seed = 2
	cfg.Params.GrassPatchCount = 0
	cfg.Params.FireSpreadChance = 1
	cfg.Params.BurnTTL = 2
	cfg.Params.GrassSpreadChance = 0
	cfg.Params.ShrubGrowthChance = 0
	cfg.Params.TreeGrowthChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.vegCurr[0] = VegetationGrass
	world.vegCurr[1] = VegetationGrass
	copy(world.vegNext, world.vegCurr)

	world.IgniteAt(0, 0)
	world.Step()

	if world.burnTTL[0] == 0 {
		t.Fatalf("source tile should still be burning, ttl=%d", world.burnTTL[0])
	}
	if world.burnTTL[1] == 0 {
		t.Fatalf("neighbor tile should ignite when spread chance is 1")
	}
	if world.vegCurr[1] != VegetationGrass {
		t.Fatalf("vegetation should remain until burn completes, got %v", world.vegCurr[1])
	}
}

func TestRainRegionRasterizesAndExpires(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 11
	cfg.Height = 11
	cfg.Params.RainSpawnChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.rainRegions = append(world.rainRegions, rainRegion{
		cx:                 5.5,
		cy:                 5.5,
		radiusX:            3.2,
		radiusY:            3.2,
		baseStrength:       1,
		strength:           1,
		strengthVariation:  0,
		targetBaseStrength: 1,
		targetRadiusX:      3.2,
		targetRadiusY:      3.2,
		ttl:                2,
		maxTTL:             2,
		falloff:            1.15,
		noiseScale:         0.01,
		noiseStretchX:      1,
		noiseStretchY:      1,
		threshold:          0,
		preset:             rainPresetPuffy,
	})

	world.updateRainMask()

	centerIdx := 5*world.w + 5
	if got := world.rainCurr[centerIdx]; got < 0.6 {
		t.Fatalf("expected strong rain at center, got %.3f", got)
	}

	edgeIdx := 5*world.w + 8
	if world.rainCurr[edgeIdx] >= world.rainCurr[centerIdx] {
		t.Fatalf("expected gaussian falloff, edge %.3f center %.3f", world.rainCurr[edgeIdx], world.rainCurr[centerIdx])
	}

	outsideIdx := 5*world.w + 0
	if world.rainCurr[outsideIdx] > 0.3 {
		t.Fatalf("expected mask edge to stay soft, got %.3f", world.rainCurr[outsideIdx])
	}

	if len(world.rainRegions) != 1 {
		t.Fatalf("expected region to persist with ttl decrement, len=%d", len(world.rainRegions))
	}
	if world.rainRegions[0].ttl != 1 {
		t.Fatalf("expected ttl to decrement to 1, got %d", world.rainRegions[0].ttl)
	}

	world.updateRainMask()
	if len(world.rainRegions) != 0 {
		t.Fatalf("expected region to expire after second tick, len=%d", len(world.rainRegions))
	}
	maxVal := float32(0)
	for _, v := range world.rainCurr {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal < 0.55 {
		t.Fatalf("expected second tick to still render rain, max=%.3f", maxVal)
	}

	world.updateRainMask()
	for i, v := range world.rainCurr {
		if v != 0 {
			t.Fatalf("expected rain mask to clear after expiry, idx=%d val=%.3f", i, v)
		}
	}
}

func TestSpawnRainRespectsCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 8
	cfg.Height = 8
	cfg.Params.RainSpawnChance = 1
	cfg.Params.RainMaxRegions = 2
	cfg.Params.RainTTLMin = 1
	cfg.Params.RainTTLMax = 1

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.spawnRainRegion()
	world.spawnRainRegion()
	world.spawnRainRegion()

	if len(world.rainRegions) != 2 {
		t.Fatalf("expected rain regions capped at 2, got %d", len(world.rainRegions))
	}

	world.rainRegions = []rainRegion{
		{
			cx:                 2,
			cy:                 2,
			radiusX:            2.5,
			radiusY:            2.5,
			baseStrength:       1,
			strength:           1,
			targetBaseStrength: 1,
			targetRadiusX:      2.5,
			targetRadiusY:      2.5,
			ttl:                1,
			maxTTL:             1,
			falloff:            1.1,
			noiseScale:         0.05,
			noiseStretchX:      1,
			noiseStretchY:      1,
			threshold:          0.2,
			preset:             rainPresetPuffy,
		},
		{
			cx:                 5,
			cy:                 5,
			radiusX:            2.5,
			radiusY:            2.5,
			baseStrength:       1,
			strength:           1,
			targetBaseStrength: 1,
			targetRadiusX:      2.5,
			targetRadiusY:      2.5,
			ttl:                1,
			maxTTL:             1,
			falloff:            1.1,
			noiseScale:         0.05,
			noiseStretchX:      1,
			noiseStretchY:      1,
			threshold:          0.2,
			preset:             rainPresetPuffy,
		},
	}

	world.updateRainMask()
	if len(world.rainRegions) != 0 {
		t.Fatalf("expected regions to expire after ttl, got %d", len(world.rainRegions))
	}

	world.spawnRainRegion()
	if len(world.rainRegions) != 1 {
		t.Fatalf("expected new rain region after expiry, got %d", len(world.rainRegions))
	}
}

func TestRebuildDisplayEncodesVegetationAndBurning(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 2
	cfg.Height = 1
	cfg.Params.GrassPatchCount = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.groundCurr[0] = GroundDirt
	world.vegCurr[0] = VegetationGrass
	world.burnTTL[0] = 0

	world.rebuildDisplay()
	expectedGrass := encodeDisplayValue(GroundDirt, VegetationGrass, false)
	if world.display[0] != expectedGrass {
		t.Fatalf("expected grass to influence display, want %d got %d", expectedGrass, world.display[0])
	}

	world.burnTTL[0] = 2
	world.rebuildDisplay()
	expectedBurn := encodeDisplayValue(GroundDirt, VegetationGrass, true)
	if world.display[0] != expectedBurn {
		t.Fatalf("expected burning flag to set display bit, want %d got %d", expectedBurn, world.display[0])
	}

	world.groundCurr[1] = GroundRock
	world.vegCurr[1] = VegetationNone
	world.burnTTL[1] = 0
	world.rebuildDisplay()
	if world.display[1] != uint8(GroundRock) {
		t.Fatalf("expected bare rock to preserve ground encoding, got %d", world.display[1])
	}
}

func TestPaletteProvidesDistinctEntries(t *testing.T) {
	world := NewWithConfig(DefaultConfig())
	palette := world.Palette()

	dirtIdx := encodeDisplayValue(GroundDirt, VegetationNone, false)
	grassIdx := encodeDisplayValue(GroundDirt, VegetationGrass, false)
	burnIdx := encodeDisplayValue(GroundDirt, VegetationGrass, true)

	if len(palette) <= int(burnIdx) {
		t.Fatalf("palette too small, need at least %d entries got %d", burnIdx+1, len(palette))
	}

	dirt := color.NRGBA{R: palette[dirtIdx].R, G: palette[dirtIdx].G, B: palette[dirtIdx].B, A: palette[dirtIdx].A}
	grass := color.NRGBA{R: palette[grassIdx].R, G: palette[grassIdx].G, B: palette[grassIdx].B, A: palette[grassIdx].A}
	burn := color.NRGBA{R: palette[burnIdx].R, G: palette[burnIdx].G, B: palette[burnIdx].B, A: palette[burnIdx].A}

	if dirt == grass {
		t.Fatalf("expected vegetation palette entry to differ from bare ground: %v vs %v", dirt, grass)
	}
	if burn == grass {
		t.Fatalf("expected burning palette entry to differ from vegetation: %v vs %v", burn, grass)
	}
}

func TestVegetationSpreadFromSeed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 4
	cfg.Height = 4
	cfg.Seed = 2024
	cfg.Params.GrassPatchCount = 0
	cfg.Params.GrassSpreadChance = 1
	cfg.Params.GrassNeighborThreshold = 1
	cfg.Params.ShrubGrowthChance = 0
	cfg.Params.TreeGrowthChance = 0
	cfg.Params.RockChance = 0
	cfg.Params.RainSpawnChance = 0
	cfg.Params.VolcanoProtoSpawnChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	// Seed a single grass tile in the center.
	idx := 5 // (1,1)
	world.vegCurr[idx] = VegetationGrass
	copy(world.vegNext, world.vegCurr)

	world.Step()

	// Adjacent dirt tiles should have become grass.
	expectedGrass := []int{0, 1, 2, 4, 6, 8, 9, 10}
	for _, pos := range expectedGrass {
		if world.vegCurr[pos] != VegetationGrass {
			t.Fatalf("expected vegetation at %d to spread to grass, got %v", pos, world.vegCurr[pos])
		}
	}
}

func TestVegetationSuccessionDeterministic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 4
	cfg.Height = 4
	cfg.Seed = 77
	cfg.Params.GrassPatchCount = 0
	cfg.Params.GrassSpreadChance = 1
	cfg.Params.ShrubGrowthChance = 1
	cfg.Params.TreeGrowthChance = 1

	worldA := NewWithConfig(cfg)
	worldA.Reset(0)

	// Build deterministic starting layout.
	layout := map[int]Vegetation{
		1:  VegetationGrass,
		4:  VegetationGrass,
		5:  VegetationGrass,
		6:  VegetationGrass,
		9:  VegetationShrub,
		10: VegetationShrub,
		11: VegetationShrub,
		14: VegetationShrub,
	}
	for idx, v := range layout {
		worldA.vegCurr[idx] = v
	}
	copy(worldA.vegNext, worldA.vegCurr)

	worldB := NewWithConfig(cfg)
	worldB.Reset(0)
	for idx, v := range layout {
		worldB.vegCurr[idx] = v
	}
	copy(worldB.vegNext, worldB.vegCurr)

	worldA.Step()
	worldB.Step()

	if !slices.Equal(worldA.vegCurr, worldB.vegCurr) {
		t.Fatal("vegetation succession diverged for identical seeds")
	}

	if worldA.vegCurr[5] != VegetationShrub {
		t.Fatalf("expected tile 5 to advance to shrub, got %v", worldA.vegCurr[5])
	}
	if worldA.vegCurr[10] != VegetationTree {
		t.Fatalf("expected tile 10 to advance to tree, got %v", worldA.vegCurr[10])
	}
}

func TestVegetationMetricsGrowthCurve(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 6
	cfg.Height = 6
	cfg.Seed = 1234
	cfg.Params.RockChance = 0
	cfg.Params.GrassPatchCount = 0
	cfg.Params.GrassSpreadChance = 1
	cfg.Params.GrassNeighborThreshold = 1
	cfg.Params.ShrubGrowthChance = 0
	cfg.Params.TreeGrowthChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	center := (cfg.Height/2)*cfg.Width + (cfg.Width / 2)
	world.vegCurr[center] = VegetationGrass
	copy(world.vegNext, world.vegCurr)

	const steps = 3
	metrics := make([]VegetationMetrics, 0, steps)
	for i := 0; i < steps; i++ {
		world.Step()
		metrics = append(metrics, world.Metrics())
	}

	for i := 1; i < len(metrics); i++ {
		if metrics[i].TotalVegetated <= metrics[i-1].TotalVegetated {
			t.Fatalf("vegetated tiles should increase early; step %d: %d <= %d", i, metrics[i].TotalVegetated, metrics[i-1].TotalVegetated)
		}
	}

	if len(metrics) == 0 {
		t.Fatal("expected metrics to be recorded")
	}

	hist := metrics[len(metrics)-1].ClusterHistogram
	hasCluster := false
	for size := 2; size < len(hist); size++ {
		if hist[size] > 0 {
			hasCluster = true
			break
		}
	}
	if !hasCluster {
		t.Fatalf("expected at least one cluster larger than size 1, histogram=%v", hist)
	}
}

func countGroundTiles(buf []Ground, target Ground) int {
	count := 0
	for _, v := range buf {
		if v == target {
			count++
		}
	}
	return count
}

type rainTelemetry struct {
	LavaMean          float64
	LavaMax           int
	LavaMin           int
	BurningMean       float64
	RainMean          float64
	RainCoverageMean  float64
	VegetationMean    float64
	ActiveRainRegions float64
}

func TestWindVectorAtZeroNoiseScale(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Params.WindNoiseScale = 0
	cfg.Params.WindSpeedScale = 0.8

	world := NewWithConfig(cfg)

	vx, vy := world.WindVectorAt(10.5, 12.5)
	if vx != 0 || vy != 0 {
		t.Fatalf("expected calm wind when noise scale is zero, got (%0.4f,%0.4f)", vx, vy)
	}
}

func TestWindVectorAtDeterministicAndBounded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Seed = 1357
	cfg.Params.WindNoiseScale = 0.018
	cfg.Params.WindSpeedScale = 1.2

	world := NewWithConfig(cfg)

	vx1, vy1 := world.WindVectorAt(24.5, 18.5)
	vx2, vy2 := world.WindVectorAt(24.5, 18.5)
	if vx1 != vx2 || vy1 != vy2 {
		t.Fatalf("expected deterministic sample, got (%0.4f,%0.4f) vs (%0.4f,%0.4f)", vx1, vy1, vx2, vy2)
	}

	vxOther, vyOther := world.WindVectorAt(56.5, 18.5)
	delta := math.Hypot(vx1-vxOther, vy1-vyOther)
	if delta < 1e-4 {
		t.Fatalf("expected spatial variation in wind vector, delta=%0.6f", delta)
	}

	speed := math.Hypot(vx1, vy1)
	expected := cfg.Params.WindSpeedScale
	if speed > expected+1e-6 {
		t.Fatalf("expected speed <= %0.3f, got %0.3f", expected, speed)
	}
	if speed == 0 {
		t.Fatalf("expected non-zero wind speed, got 0")
	}
}

func TestRainRegionsShareGlobalWindField(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Seed = 4242
	cfg.Params.WindNoiseScale = 0.02
	cfg.Params.WindSpeedScale = 0.7

	world := NewWithConfig(cfg)

	regionA := rainRegion{cx: 24.5, cy: 18.5, ttl: 10}
	regionB := regionA
	regionB.noiseSeed = 99

	regions := []rainRegion{regionA, regionB}
	updatedA := world.advanceRainRegion(regions, 0)
	updatedB := world.advanceRainRegion(regions, 1)

	deltaVX := math.Abs(updatedA.vx - updatedB.vx)
	deltaVY := math.Abs(updatedA.vy - updatedB.vy)
	if deltaVX > 1e-6 || deltaVY > 1e-6 {
		t.Fatalf("expected shared wind velocity, got A(%0.6f,%0.6f) vs B(%0.6f,%0.6f)", updatedA.vx, updatedA.vy, updatedB.vx, updatedB.vy)
	}

	targetVX, targetVY := world.WindVectorAt(regionA.cx, regionA.cy)
	if targetVX != 0 || targetVY != 0 {
		const inertia = 0.08
		const cohesionBlend = 0.08
		expectedVX := targetVX * inertia * (1 - cohesionBlend)
		expectedVY := targetVY * inertia * (1 - cohesionBlend)
		if math.Abs(updatedA.vx-expectedVX) > 1e-6 || math.Abs(updatedA.vy-expectedVY) > 1e-6 {
			t.Fatalf("expected easing toward (%0.6f,%0.6f), got (%0.6f,%0.6f)", expectedVX, expectedVY, updatedA.vx, updatedA.vy)
		}
	}
}
