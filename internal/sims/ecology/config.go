package ecology

import "strconv"

// Params holds tunable thresholds and probabilities for the ecology sim.
type Params struct {
	RockChance          float64
	GrassPatchCount     int
	GrassPatchRadiusMin int
	GrassPatchRadiusMax int
	GrassPatchDensity   float64

	LavaLifeMin int
	LavaLifeMax int
	BurnTTL     int
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
			RockChance:          0.05,
			GrassPatchCount:     12,
			GrassPatchRadiusMin: 2,
			GrassPatchRadiusMax: 5,
			GrassPatchDensity:   0.6,
			LavaLifeMin:         12,
			LavaLifeMax:         32,
			BurnTTL:             3,
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
	if v, ok := cfg["burn_ttl"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			c.Params.BurnTTL = parsed
		}
	}
	return c
}
