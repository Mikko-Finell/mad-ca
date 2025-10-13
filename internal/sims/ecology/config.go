package ecology

import "strconv"

// Params holds tunable thresholds and probabilities for the ecology sim.
type Params struct {
	RockChance          float64
	GrassPatchCount     int
	GrassPatchRadiusMin int
	GrassPatchRadiusMax int
	GrassPatchDensity   float64

	LavaLifeMin         int
	LavaLifeMax         int
	LavaSpreadChance    float64
	LavaSpreadMaskFloor float64
	LavaCoolingExtra    float64
	BurnTTL             int

	FireSpreadChance         float64
	FireLavaIgniteChance     float64
	FireRainSpreadDampen     float64
	FireRainExtinguishChance float64

	GrassNeighborThreshold int
	GrassSpreadChance      float64
	ShrubNeighborThreshold int
	ShrubGrowthChance      float64
	TreeNeighborThreshold  int
	TreeGrowthChance       float64

	VolcanoProtoMaxRegions        int
	VolcanoProtoSpawnChance       float64
	VolcanoProtoTectonicThreshold float64
	VolcanoProtoRadiusMin         int
	VolcanoProtoRadiusMax         int
	VolcanoProtoTTLMin            int
	VolcanoProtoTTLMax            int
	VolcanoProtoStrengthMin       float64
	VolcanoProtoStrengthMax       float64
	VolcanoUpliftChanceBase       float64
	VolcanoEruptionChanceBase     float64
}

// Config controls the Ecology simulation dimensions.
type Config struct {
	Width  int
	Height int

	Seed int64

	Params Params
}

// DefaultConfig returns the standard configuration.
func DefaultConfig() Config {
	return Config{
		Width:  256,
		Height: 256,
		Seed:   1337,
		Params: Params{
			RockChance:                    0.05,
			GrassPatchCount:               12,
			GrassPatchRadiusMin:           2,
			GrassPatchRadiusMax:           5,
			GrassPatchDensity:             0.6,
			LavaLifeMin:                   12,
			LavaLifeMax:                   32,
			LavaSpreadChance:              0.08,
			LavaSpreadMaskFloor:           0.2,
			LavaCoolingExtra:              1,
			BurnTTL:                       3,
			FireSpreadChance:              0.25,
			FireLavaIgniteChance:          0.8,
			FireRainSpreadDampen:          0.75,
			FireRainExtinguishChance:      0.5,
			GrassNeighborThreshold:        1,
			GrassSpreadChance:             0.25,
			ShrubNeighborThreshold:        3,
			ShrubGrowthChance:             0.04,
			TreeNeighborThreshold:         3,
			TreeGrowthChance:              0.02,
			VolcanoProtoMaxRegions:        6,
			VolcanoProtoSpawnChance:       0.02,
			VolcanoProtoTectonicThreshold: 0.6,
			VolcanoProtoRadiusMin:         10,
			VolcanoProtoRadiusMax:         22,
			VolcanoProtoTTLMin:            10,
			VolcanoProtoTTLMax:            25,
			VolcanoProtoStrengthMin:       0.4,
			VolcanoProtoStrengthMax:       0.9,
			VolcanoUpliftChanceBase:       0.00002,
			VolcanoEruptionChanceBase:     0.00005,
		},
	}
}

// FromMap populates the config from a string map (flag-style key/value pairs).
func FromMap(cfg map[string]string) Config {
	c := DefaultConfig()
	if cfg == nil {
		return c
	}
	if v, ok := cfg["w"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			c.Width = parsed
		}
	}
	if v, ok := cfg["h"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			c.Height = parsed
		}
	}
	if v, ok := cfg["seed"]; ok {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.Seed = parsed
		}
	}
	if v, ok := cfg["rock_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.RockChance = parsed
		}
	}
	if v, ok := cfg["grass_patch_count"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.GrassPatchCount = parsed
		}
	}
	if v, ok := cfg["grass_patch_radius_min"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.GrassPatchRadiusMin = parsed
		}
	}
	if v, ok := cfg["grass_patch_radius_max"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.GrassPatchRadiusMax = parsed
		}
	}
	if c.Params.GrassPatchRadiusMax < c.Params.GrassPatchRadiusMin {
		c.Params.GrassPatchRadiusMax = c.Params.GrassPatchRadiusMin
	}
	if v, ok := cfg["grass_patch_density"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			c.Params.GrassPatchDensity = parsed
		}
	}
	if v, ok := cfg["lava_life_min"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.LavaLifeMin = parsed
		}
	}
	if v, ok := cfg["lava_life_max"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.LavaLifeMax = parsed
		}
	}
	if c.Params.LavaLifeMax < c.Params.LavaLifeMin {
		c.Params.LavaLifeMax = c.Params.LavaLifeMin
	}
	if v, ok := cfg["lava_spread_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.LavaSpreadChance = parsed
		}
	}
	if v, ok := cfg["lava_spread_mask_floor"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			if parsed < 0 {
				parsed = 0
			}
			if parsed > 1 {
				parsed = 1
			}
			c.Params.LavaSpreadMaskFloor = parsed
		}
	}
	if v, ok := cfg["lava_cooling_extra"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			if parsed < 0 {
				parsed = 0
			}
			c.Params.LavaCoolingExtra = parsed
		}
	}
	if v, ok := cfg["burn_ttl"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.BurnTTL = parsed
		}
	}
	if v, ok := cfg["fire_spread_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.FireSpreadChance = parsed
		}
	}
	if v, ok := cfg["fire_lava_ignite_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.FireLavaIgniteChance = parsed
		}
	}
	if v, ok := cfg["fire_rain_spread_dampen"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.FireRainSpreadDampen = parsed
		}
	}
	if v, ok := cfg["fire_rain_extinguish_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.FireRainExtinguishChance = parsed
		}
	}
	if v, ok := cfg["grass_neighbor_threshold"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.GrassNeighborThreshold = parsed
		}
	}
	if v, ok := cfg["grass_spread_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.GrassSpreadChance = parsed
		}
	}
	if v, ok := cfg["shrub_neighbor_threshold"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.ShrubNeighborThreshold = parsed
		}
	}
	if v, ok := cfg["shrub_growth_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.ShrubGrowthChance = parsed
		}
	}
	if v, ok := cfg["tree_neighbor_threshold"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.TreeNeighborThreshold = parsed
		}
	}
	if v, ok := cfg["tree_growth_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.TreeGrowthChance = parsed
		}
	}
	if v, ok := cfg["volcano_proto_max_regions"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.VolcanoProtoMaxRegions = parsed
		}
	}
	if v, ok := cfg["volcano_proto_spawn_chance"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.VolcanoProtoSpawnChance = parsed
		}
	}
	if v, ok := cfg["volcano_proto_tectonic_threshold"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			c.Params.VolcanoProtoTectonicThreshold = parsed
		}
	}
	if v, ok := cfg["volcano_proto_radius_min"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.VolcanoProtoRadiusMin = parsed
		}
	}
	if v, ok := cfg["volcano_proto_radius_max"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.VolcanoProtoRadiusMax = parsed
		}
	}
	if c.Params.VolcanoProtoRadiusMax < c.Params.VolcanoProtoRadiusMin {
		c.Params.VolcanoProtoRadiusMax = c.Params.VolcanoProtoRadiusMin
	}
	if v, ok := cfg["volcano_proto_ttl_min"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.VolcanoProtoTTLMin = parsed
		}
	}
	if v, ok := cfg["volcano_proto_ttl_max"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.VolcanoProtoTTLMax = parsed
		}
	}
	if c.Params.VolcanoProtoTTLMax < c.Params.VolcanoProtoTTLMin {
		c.Params.VolcanoProtoTTLMax = c.Params.VolcanoProtoTTLMin
	}
	if v, ok := cfg["volcano_proto_strength_min"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			c.Params.VolcanoProtoStrengthMin = parsed
		}
	}
	if v, ok := cfg["volcano_proto_strength_max"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			c.Params.VolcanoProtoStrengthMax = parsed
		}
	}
	if c.Params.VolcanoProtoStrengthMax < c.Params.VolcanoProtoStrengthMin {
		c.Params.VolcanoProtoStrengthMax = c.Params.VolcanoProtoStrengthMin
	}
	if v, ok := cfg["volcano_uplift_chance_base"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.VolcanoUpliftChanceBase = parsed
		}
	}
	if v, ok := cfg["volcano_eruption_chance_base"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			c.Params.VolcanoEruptionChanceBase = parsed
		}
	}
	return c
}
