package ecology

import (
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

func TestApplyEruptionsSeedsLavaAndMountains(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 9
	cfg.Height = 9
	cfg.Params.GrassPatchCount = 0
	cfg.Params.VolcanoProtoSpawnChance = 0
	cfg.Params.VolcanoEruptionChanceBase = 10
	cfg.Params.LavaLifeMin = 4
	cfg.Params.LavaLifeMax = 4

	world := NewWithConfig(cfg)
	world.Reset(0)
	world.rng.Seed(1)

	for i := range world.groundCurr {
		world.groundCurr[i] = GroundRock
		world.display[i] = uint8(GroundRock)
	}

	world.volcanoRegions = []volcanoProtoRegion{{
		cx:       4.5,
		cy:       4.5,
		radius:   3,
		strength: 1,
		ttl:      1,
	}}

	world.updateVolcanoMask()

	if len(world.expiredVolcanoProtos) != 1 {
		t.Fatalf("expected expired proto to be tracked, got %d", len(world.expiredVolcanoProtos))
	}

	world.applyEruptions()

	centerIdx := 4*world.w + 4
	if world.groundCurr[centerIdx] != GroundLava {
		t.Fatalf("expected eruption core to produce lava, got %v", world.groundCurr[centerIdx])
	}
	if world.lavaLife[centerIdx] != 4 {
		t.Fatalf("expected lava life of 4, got %d", world.lavaLife[centerIdx])
	}

	rimIdx := 2*world.w + 4
	if world.groundCurr[rimIdx] != GroundMountain {
		t.Fatalf("expected rim uplift to convert to mountain, got %v", world.groundCurr[rimIdx])
	}
	if world.lavaLife[rimIdx] != 0 {
		t.Fatalf("rim tile should not retain lava life, got %d", world.lavaLife[rimIdx])
	}

	if len(world.expiredVolcanoProtos) != 0 {
		t.Fatalf("expected eruption consumption to clear expired list, got %d", len(world.expiredVolcanoProtos))
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

func TestLavaIgnitesAdjacentVegetation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 1
	cfg.Seed = 3
	cfg.Params.GrassPatchCount = 0
	cfg.Params.FireLavaIgniteChance = 1
	cfg.Params.FireSpreadChance = 0
	cfg.Params.LavaSpreadChance = 0
	cfg.Params.GrassSpreadChance = 0
	cfg.Params.ShrubGrowthChance = 0
	cfg.Params.TreeGrowthChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.groundCurr[1] = GroundLava
	world.lavaLife[1] = 5
	world.vegCurr[0] = VegetationShrub
	copy(world.vegNext, world.vegCurr)

	world.Step()

	if world.burnTTL[0] == 0 {
		t.Fatalf("expected lava-adjacent vegetation to ignite")
	}
}

func TestRainPreventsLavaIgnitionWhenFullyWet(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 1
	cfg.Seed = 4
	cfg.Params.GrassPatchCount = 0
	cfg.Params.FireLavaIgniteChance = 1
	cfg.Params.FireSpreadChance = 0
	cfg.Params.FireRainSpreadDampen = 1
	cfg.Params.LavaSpreadChance = 0
	cfg.Params.GrassSpreadChance = 0
	cfg.Params.ShrubGrowthChance = 0
	cfg.Params.TreeGrowthChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.groundCurr[1] = GroundLava
	world.lavaLife[1] = 5
	world.vegCurr[0] = VegetationTree
	world.rainCurr[0] = 1
	copy(world.vegNext, world.vegCurr)

	world.Step()

	if world.burnTTL[0] != 0 {
		t.Fatalf("rain dampening should prevent lava ignition, ttl=%d", world.burnTTL[0])
	}
}

func TestLavaCoolsToRock(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 1
	cfg.Height = 1
	cfg.Seed = 11
	cfg.Params.GrassPatchCount = 0
	cfg.Params.LavaLifeMin = 1
	cfg.Params.LavaLifeMax = 1
	cfg.Params.LavaSpreadChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.groundCurr[0] = GroundLava
	world.lavaLife[0] = 1
	world.display[0] = uint8(GroundLava)

	world.Step()

	if world.groundCurr[0] != GroundRock {
		t.Fatalf("expected lava to cool to rock, got %v", world.groundCurr[0])
	}
	if world.lavaLife[0] != 0 {
		t.Fatalf("expected lava life to clear after cooling, got %d", world.lavaLife[0])
	}
	if world.display[0] != uint8(GroundRock) {
		t.Fatalf("display buffer should reflect cooled rock, got %d", world.display[0])
	}
}

func TestLavaSpreadFeedsFire(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 4
	cfg.Height = 1
	cfg.Seed = 21
	cfg.Params.GrassPatchCount = 0
	cfg.Params.FireLavaIgniteChance = 1
	cfg.Params.FireSpreadChance = 0
	cfg.Params.LavaSpreadChance = 1
	cfg.Params.LavaSpreadMaskFloor = 1
	cfg.Params.LavaLifeMin = 2
	cfg.Params.LavaLifeMax = 2
	cfg.Params.GrassSpreadChance = 0
	cfg.Params.ShrubGrowthChance = 0
	cfg.Params.TreeGrowthChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.groundCurr[1] = GroundLava
	world.lavaLife[1] = 3
	world.vegCurr[3] = VegetationGrass
	copy(world.vegNext, world.vegCurr)

	world.Step()

	if world.groundCurr[2] != GroundLava {
		t.Fatalf("expected lava to spread to neighbor, got %v", world.groundCurr[2])
	}
	if world.lavaLife[2] != 2 {
		t.Fatalf("expected new lava life of 2, got %d", world.lavaLife[2])
	}
	if world.vegCurr[2] != VegetationNone {
		t.Fatalf("lava should clear vegetation on takeover, got %v", world.vegCurr[2])
	}
	if world.burnTTL[3] == 0 {
		t.Fatalf("new lava should ignite adjacent vegetation, burn ttl=%d", world.burnTTL[3])
	}
	if world.vegCurr[3] != VegetationGrass {
		t.Fatalf("vegetation should persist until burn completes, got %v", world.vegCurr[3])
	}
}

func TestLavaSpreadRequiresMaskOrFloor(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 1
	cfg.Seed = 31
	cfg.Params.GrassPatchCount = 0
	cfg.Params.LavaSpreadChance = 1
	cfg.Params.LavaSpreadMaskFloor = 0
	cfg.Params.LavaCoolingExtra = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.groundCurr[1] = GroundLava
	world.lavaLife[1] = 4
	world.volCurr[0] = 0
	world.volCurr[1] = 0

	world.applyLava()

	if world.groundCurr[0] == GroundLava {
		t.Fatal("lava should not spread without mask influence or floor")
	}

	cfg.Params.LavaSpreadMaskFloor = 1
	world = NewWithConfig(cfg)
	world.Reset(0)
	world.groundCurr[1] = GroundLava
	world.lavaLife[1] = 4
	world.volCurr[0] = 0
	world.volCurr[1] = 0

	world.applyLava()

	if world.groundCurr[0] != GroundLava {
		t.Fatal("lava should spread when floor guarantees chance")
	}
}

func TestLavaCoolingAcceleratesWhenMaskLow(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 1
	cfg.Height = 1
	cfg.Seed = 41
	cfg.Params.GrassPatchCount = 0
	cfg.Params.LavaSpreadChance = 0
	cfg.Params.LavaCoolingExtra = 2

	world := NewWithConfig(cfg)
	world.Reset(0)
	world.groundCurr[0] = GroundLava
	world.lavaLife[0] = 5
	world.volCurr[0] = 0

	world.applyLava()

	if got := int(world.lavaLife[0]); got != 2 {
		t.Fatalf("expected accelerated cooling outside mask, got lava life %d", got)
	}

	world = NewWithConfig(cfg)
	world.Reset(0)
	world.groundCurr[0] = GroundLava
	world.lavaLife[0] = 5
	world.volCurr[0] = 1

	world.applyLava()

	if got := int(world.lavaLife[0]); got != 4 {
		t.Fatalf("expected minimal cooling under mask, got lava life %d", got)
	}
}

func TestLavaCoolingAcceleratesWithRain(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 1
	cfg.Height = 1
	cfg.Seed = 42
	cfg.Params.GrassPatchCount = 0
	cfg.Params.LavaSpreadChance = 0
	cfg.Params.LavaCoolingExtra = 0

	world := NewWithConfig(cfg)
	world.Reset(0)
	world.groundCurr[0] = GroundLava
	world.lavaLife[0] = 10
	world.rainCurr[0] = 0.5

	world.applyLava()

	if got := int(world.lavaLife[0]); got != 5 {
		t.Fatalf("expected rain to add cooling bonus, lava life=%d", got)
	}

	world.Reset(0)
	world.groundCurr[0] = GroundLava
	world.lavaLife[0] = 10
	world.rainCurr[0] = 1
	world.volCurr[0] = 1

	world.applyLava()

	if got := int(world.lavaLife[0]); got != 1 {
		t.Fatalf("expected full rain to nearly extinguish lava, lava life=%d", got)
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

func TestVolcanoCyclesRegression(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 48
	cfg.Height = 48
	cfg.Seed = 9001
	cfg.Params.RockChance = 0.02
	cfg.Params.GrassPatchCount = 0
	cfg.Params.GrassSpreadChance = 0
	cfg.Params.ShrubGrowthChance = 0
	cfg.Params.TreeGrowthChance = 0
	cfg.Params.FireSpreadChance = 0
	cfg.Params.FireLavaIgniteChance = 0
	cfg.Params.BurnTTL = 1
	cfg.Params.VolcanoProtoMaxRegions = 1
	cfg.Params.VolcanoProtoSpawnChance = 0.3
	cfg.Params.VolcanoProtoTTLMin = 12
	cfg.Params.VolcanoProtoTTLMax = 12
	cfg.Params.VolcanoProtoRadiusMin = 12
	cfg.Params.VolcanoProtoRadiusMax = 12
	cfg.Params.VolcanoProtoStrengthMin = 0.85
	cfg.Params.VolcanoProtoStrengthMax = 0.85
	cfg.Params.VolcanoUpliftChanceBase = 0.06
	cfg.Params.VolcanoEruptionChanceBase = 1
	cfg.Params.LavaLifeMin = 10
	cfg.Params.LavaLifeMax = 14
	cfg.Params.LavaSpreadChance = 0.22
	cfg.Params.LavaSpreadMaskFloor = 0.15
	cfg.Params.LavaCoolingExtra = 1.2

	world := NewWithConfig(cfg)
	world.Reset(0)

	const steps = 600
	lavaHistory := make([]int, steps)
	mountainHistory := make([]int, steps)

	for i := 0; i < steps; i++ {
		applyOscillatingRain(world, i)
		world.Step()
		lavaHistory[i] = countGroundTiles(world.groundCurr, GroundLava)
		mountainHistory[i] = countGroundTiles(world.groundCurr, GroundMountain)
	}

	checkpoints := []struct {
		tick     int
		lava     int
		mountain int
	}{
		{tick: 90, lava: 66, mountain: 10},
		{tick: 179, lava: 23, mountain: 282},
		{tick: 359, lava: 88, mountain: 705},
		{tick: 419, lava: 33, mountain: 803},
		{tick: steps - 1, lava: 0, mountain: 933},
	}

	for _, checkpoint := range checkpoints {
		if lavaHistory[checkpoint.tick] != checkpoint.lava {
			t.Fatalf("lava count mismatch at tick %d: expected %d, got %d", checkpoint.tick, checkpoint.lava, lavaHistory[checkpoint.tick])
		}
		if mountainHistory[checkpoint.tick] != checkpoint.mountain {
			t.Fatalf("mountain count mismatch at tick %d: expected %d, got %d", checkpoint.tick, checkpoint.mountain, mountainHistory[checkpoint.tick])
		}
	}

	maxLava := 0
	minLava := lavaHistory[0]
	for _, v := range lavaHistory {
		if v > maxLava {
			maxLava = v
		}
		if v < minLava {
			minLava = v
		}
	}

	if maxLava != 163 {
		t.Fatalf("expected lava peak of 163 tiles, got %d", maxLava)
	}
	if minLava != 0 {
		t.Fatalf("expected lava trough to fully cool, min=%d", minLava)
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

func applyOscillatingRain(world *World, tick int) {
	width := world.w
	height := world.h
	if width == 0 || height == 0 {
		return
	}
	bandRadius := int(math.Max(4, float64(width)/8))
	sweep := width + bandRadius*2
	center := (tick * 3) % sweep
	center -= bandRadius
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			dist := math.Abs(float64(x - center))
			if dist > float64(bandRadius) {
				world.rainCurr[idx] = 0
				continue
			}
			intensity := 1 - dist/float64(bandRadius)
			if intensity < 0 {
				intensity = 0
			}
			if intensity > 1 {
				intensity = 1
			}
			world.rainCurr[idx] = float32(intensity)
		}
	}
}
