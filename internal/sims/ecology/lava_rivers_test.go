package ecology

import (
	"math"
	"testing"
)

func TestEruptionSeedsLavaRivers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 9
	cfg.Height = 9
	cfg.Params.GrassPatchCount = 0
	cfg.Params.VolcanoProtoSpawnChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	for i := range world.groundCurr {
		world.groundCurr[i] = GroundRock
		world.display[i] = uint8(GroundRock)
	}

	region := volcanoProtoRegion{
		cx:       4.5,
		cy:       4.5,
		radius:   3,
		strength: 1,
		ttl:      0,
	}

	world.eruptRegion(region)

	centerIdx := 4*world.w + 4
	if world.groundCurr[centerIdx] != GroundLava {
		t.Fatalf("expected eruption core to produce lava, got %v", world.groundCurr[centerIdx])
	}
	if h := world.lavaHeight[centerIdx]; h < 2 || h > 3 {
		t.Fatalf("expected lava thickness 2-3, got %d", h)
	}
	if temp := world.lavaTemp[centerIdx]; math.Abs(float64(temp-1)) > 1e-6 {
		t.Fatalf("expected hot lava temp of 1, got %.3f", temp)
	}
	if world.lavaTip[centerIdx] {
		t.Fatal("caldera core should not start as a tip")
	}

	rimIdx := 2*world.w + 4
	if world.groundCurr[rimIdx] != GroundMountain {
		t.Fatalf("expected rim uplift to convert to mountain, got %v", world.groundCurr[rimIdx])
	}

	for i, v := range world.lavaChannel {
		if v != 0 {
			t.Fatalf("expected channels to reset on eruption, idx=%d value=%.3f", i, v)
		}
	}

	if len(world.lavaVents) == 0 {
		t.Fatal("expected eruption to create active vents")
	}
	for _, vent := range world.lavaVents {
		if vent.fuel < cfg.Params.LavaLifeMin || vent.fuel > cfg.Params.LavaLifeMax {
			t.Fatalf("vent fuel out of range: %d", vent.fuel)
		}
		if vent.outIdx < 0 || vent.outIdx >= len(world.groundCurr) {
			t.Fatalf("vent out index out of bounds: %d", vent.outIdx)
		}
	}
}

func TestLavaLifeParametersClampActiveVents(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 9
	cfg.Height = 9
	cfg.Params.GrassPatchCount = 0
	cfg.Params.VolcanoProtoSpawnChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	for i := range world.groundCurr {
		world.groundCurr[i] = GroundRock
		world.display[i] = uint8(GroundRock)
	}

	region := volcanoProtoRegion{
		cx:       4.5,
		cy:       4.5,
		radius:   3,
		strength: 1,
		ttl:      0,
	}

	world.eruptRegion(region)

	if len(world.lavaVents) == 0 {
		t.Fatal("expected eruption to create active vents")
	}

	if !world.SetIntParameter("lava_life_min", 5) {
		t.Fatal("failed to set lava life min")
	}
	if !world.SetIntParameter("lava_life_max", 5) {
		t.Fatal("failed to set lava life max")
	}
	for _, vent := range world.lavaVents {
		if vent.fuel != 5 {
			t.Fatalf("expected vent fuel to clamp to new range, got %d", vent.fuel)
		}
	}

	if !world.SetIntParameter("lava_life_max", 400) {
		t.Fatal("failed to expand lava life max")
	}
	if !world.SetIntParameter("lava_life_min", 300) {
		t.Fatal("failed to expand lava life min")
	}
	for _, vent := range world.lavaVents {
		if vent.fuel < 300 || vent.fuel > 400 {
			t.Fatalf("expected vent fuel within expanded range, got %d", vent.fuel)
		}
	}
}

func TestLavaCoolingCrustsAndSolidifies(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 1
	cfg.Height = 1
	cfg.Params.GrassPatchCount = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	world.groundCurr[0] = GroundLava
	world.lavaHeight[0] = 3
	world.lavaTemp[0] = 1
	world.lavaDir[0] = -1
	world.lavaTip[0] = false
	world.lavaElevation[0] = 3

	for i := 0; i < 80; i++ {
		world.applyLava()
	}

	if world.groundCurr[0] != GroundRock {
		t.Fatalf("expected lava to cool into rock, got %v", world.groundCurr[0])
	}
	if world.lavaHeight[0] != 0 {
		t.Fatalf("expected lava thickness to clear, got %d", world.lavaHeight[0])
	}
	if world.lavaTemp[0] != 0 {
		t.Fatalf("expected lava temp to reset, got %.3f", world.lavaTemp[0])
	}
}

func TestRainAcceleratesLavaCooling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 1
	cfg.Height = 1
	cfg.Params.GrassPatchCount = 0

	dry := NewWithConfig(cfg)
	dry.Reset(0)
	dry.groundCurr[0] = GroundLava
	dry.lavaHeight[0] = 3
	dry.lavaTemp[0] = 1
	dry.lavaDir[0] = -1
	dry.lavaElevation[0] = 3
	dry.applyLava()
	dryTemp := dry.lavaTemp[0]

	rainy := NewWithConfig(cfg)
	rainy.Reset(0)
	rainy.groundCurr[0] = GroundLava
	rainy.lavaHeight[0] = 3
	rainy.lavaTemp[0] = 1
	rainy.lavaDir[0] = -1
	rainy.lavaElevation[0] = 3
	rainy.rainCurr[0] = 1
	rainy.applyLava()
	rainyTemp := rainy.lavaTemp[0]

	if rainyTemp >= dryTemp {
		t.Fatalf("expected rain to cool lava faster, dry=%.3f rainy=%.3f", dryTemp, rainyTemp)
	}
}

func TestLavaChannelReinforcement(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 1
	cfg.Params.GrassPatchCount = 0

	world := NewWithConfig(cfg)
	world.Reset(0)

	for i := range world.groundCurr {
		world.groundCurr[i] = GroundRock
	}
	world.groundCurr[1] = GroundLava
	world.lavaHeight[1] = 5
	world.lavaTemp[1] = 1
	world.lavaDir[1] = 0
	world.lavaTip[1] = true
	world.lavaForce[1] = true
	world.lavaElevation[0] = 7
	world.lavaElevation[1] = 5
	world.lavaElevation[2] = 3

	world.applyLava()

	if world.groundCurr[2] != GroundLava {
		t.Fatalf("expected forced tip to advance east, got %v", world.groundCurr[2])
	}
	expected := float32(lavaChannelGrow) * float32(1-lavaChannelDecay)
	if got := world.lavaChannel[2]; math.Abs(float64(got-expected)) > 1e-3 {
		t.Fatalf("expected channel reinforcement %.5f, got %.5f", expected, got)
	}
	if int(world.lavaHeight[1]) >= 5 {
		t.Fatalf("expected parent channel to thin, height=%d", world.lavaHeight[1])
	}
}

func TestLavaIgnitionRespectsRain(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 2
	cfg.Height = 1
	cfg.Params.GrassPatchCount = 0
	cfg.Params.FireLavaIgniteChance = 1
	cfg.Params.FireSpreadChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)
	world.groundCurr[1] = GroundLava
	world.lavaHeight[1] = 4
	world.lavaTemp[1] = 1
	world.vegCurr[0] = VegetationShrub

	world.applyFire()
	if world.burnTTL[0] == 0 {
		t.Fatalf("expected lava-adjacent vegetation to ignite")
	}

	damp := NewWithConfig(cfg)
	damp.Reset(0)
	damp.groundCurr[1] = GroundLava
	damp.lavaHeight[1] = 4
	damp.lavaTemp[1] = 1
	damp.vegCurr[0] = VegetationShrub
	damp.rainCurr[0] = 1
	damp.cfg.Params.FireRainSpreadDampen = 1

	damp.applyFire()
	if damp.burnTTL[0] > 0 {
		t.Fatalf("expected full rain to suppress lava ignition, ttl=%d", damp.burnTTL[0])
	}
}

func TestLavaTipPoolsWhenBlocked(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 2
	cfg.Height = 1
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RockChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)
	world.rng.Seed(1)

	for i := range world.groundCurr {
		world.groundCurr[i] = GroundRock
	}

	world.groundCurr[0] = GroundLava
	world.lavaHeight[0] = 2
	world.lavaTemp[0] = 0.5
	world.lavaDir[0] = 0
	world.lavaTip[0] = true
	world.lavaElevation[0] = 5
	world.lavaElevation[1] = 5

	world.applyLava()

	if world.groundCurr[1] != GroundLava {
		t.Fatalf("expected pooled lava to occupy neighbor, got %v", world.groundCurr[1])
	}
	if world.lavaDir[1] != -1 {
		t.Fatalf("expected pooled lava to have no heading, dir=%d", world.lavaDir[1])
	}
	if world.lavaTip[1] {
		t.Fatal("expected pooled lava to remain a non-tip cell")
	}
	if world.lavaHeight[1] != 1 {
		t.Fatalf("expected pooled lava to be one cell thick, got %d", world.lavaHeight[1])
	}
	if world.lavaHeight[0] != 3 {
		t.Fatalf("expected blocked tip to thicken, height=%d", world.lavaHeight[0])
	}
}

func TestLavaTipSplitsWhenFluxHigh(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Width = 3
	cfg.Height = 2
	cfg.Seed = 2
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RockChance = 0

	world := NewWithConfig(cfg)
	world.Reset(0)
	world.rng.Seed(2)

	for i := range world.groundCurr {
		world.groundCurr[i] = GroundRock
	}

	tipIdx := 0
	eastIdx := 1
	southIdx := 3
	southEastIdx := 4

	world.groundCurr[tipIdx] = GroundLava
	world.lavaHeight[tipIdx] = 5
	world.lavaTemp[tipIdx] = 1
	world.lavaDir[tipIdx] = 0
	world.lavaTip[tipIdx] = true
	world.lavaForce[tipIdx] = true

	world.lavaElevation[tipIdx] = 10
	world.lavaElevation[eastIdx] = 6
	world.lavaElevation[southIdx] = 10
	world.lavaElevation[southEastIdx] = 7

	world.applyLava()

	if world.groundCurr[eastIdx] != GroundLava {
		t.Fatalf("expected leading edge to advance east, got %v", world.groundCurr[eastIdx])
	}
	if world.groundCurr[southEastIdx] != GroundLava {
		t.Fatalf("expected high flux to split into southeast neighbor, got %v", world.groundCurr[southEastIdx])
	}
	if world.lavaDir[eastIdx] != 0 {
		t.Fatalf("expected primary child to inherit heading east, dir=%d", world.lavaDir[eastIdx])
	}
	if world.lavaDir[southEastIdx] != 1 {
		t.Fatalf("expected split child to head southeast, dir=%d", world.lavaDir[southEastIdx])
	}
	if !world.lavaTip[eastIdx] {
		t.Fatal("expected main advance to remain a tip")
	}
	if !world.lavaTip[southEastIdx] {
		t.Fatal("expected split advance to become a tip")
	}
	if world.lavaHeight[tipIdx] != 4 {
		t.Fatalf("expected parent channel to thin after spawning children, height=%d", world.lavaHeight[tipIdx])
	}
}
