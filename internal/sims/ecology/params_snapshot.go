package ecology

import (
	"strconv"

	"mad-ca/internal/core"
)

func (w *World) Parameters() core.ParameterSnapshot {
	params := w.cfg.Params
	groups := []core.ParameterGroup{
		{
			Name: "World",
			Params: []core.Parameter{
				intParam("w", "Width", w.cfg.Width),
				intParam("h", "Height", w.cfg.Height),
				int64Param("seed", "Seed", w.cfg.Seed),
			},
		},
		{
			Name: "Terrain Seeding",
			Params: []core.Parameter{
				floatParam("rock_chance", "Rock chance", params.RockChance),
				intParam("grass_patch_count", "Grass patch count", params.GrassPatchCount),
				intParam("grass_patch_radius_min", "Grass patch radius min", params.GrassPatchRadiusMin),
				intParam("grass_patch_radius_max", "Grass patch radius max", params.GrassPatchRadiusMax),
				floatParam("grass_patch_density", "Grass patch density", params.GrassPatchDensity),
			},
		},
		{
			Name: "Lava",
			Params: []core.Parameter{
				floatParam("lava_spread_chance", "Lava spread chance", params.LavaSpreadChance),
				floatParam("lava_spread_mask_floor", "Lava spread mask floor", params.LavaSpreadMaskFloor),
				floatParam("lava_flux_ref", "Lava flux reference", params.LavaFluxRef),
				floatParam("lava_cool_base", "Lava base cooling", params.LavaCoolBase),
				floatParam("lava_cool_rain", "Lava rain cooling", params.LavaCoolRain),
				floatParam("lava_cool_edge", "Lava edge cooling", params.LavaCoolEdge),
				floatParam("lava_cool_thick", "Lava thickness cooling", params.LavaCoolThick),
				floatParam("lava_cool_flux", "Lava flux cooling", params.LavaCoolFlux),
				floatParam("lava_phase_threshold", "Lava crust threshold", params.LavaPhaseThreshold),
				floatParam("lava_phase_hysteresis", "Lava thermal hysteresis", params.LavaPhaseHysteresis),
				intParam("lava_reservoir_min", "Lava reservoir min", params.LavaReservoirMin),
				intParam("lava_reservoir_max", "Lava reservoir max", params.LavaReservoirMax),
				floatParam("lava_reservoir_gain", "Lava reservoir gain", params.LavaReservoirGain),
				floatParam("lava_reservoir_head", "Lava reservoir head", params.LavaReservoirHead),
			},
		},
		{
			Name: "Fire",
			Params: []core.Parameter{
				intParam("burn_ttl", "Burn TTL", params.BurnTTL),
				floatParam("fire_spread_chance", "Fire spread chance", params.FireSpreadChance),
				floatParam("fire_lava_ignite_chance", "Fire lava ignite chance", params.FireLavaIgniteChance),
				floatParam("fire_rain_spread_dampen", "Fire rain spread dampen", params.FireRainSpreadDampen),
				floatParam("fire_rain_extinguish_chance", "Fire rain extinguish chance", params.FireRainExtinguishChance),
			},
		},
		{
			Name: "Rain",
			Params: []core.Parameter{
				intParam("rain_max_regions", "Rain max regions", params.RainMaxRegions),
				floatParam("rain_spawn_chance", "Rain spawn chance", params.RainSpawnChance),
				intParam("rain_radius_min", "Rain radius min", params.RainRadiusMin),
				intParam("rain_radius_max", "Rain radius max", params.RainRadiusMax),
				intParam("rain_ttl_min", "Rain TTL min", params.RainTTLMin),
				intParam("rain_ttl_max", "Rain TTL max", params.RainTTLMax),
				floatParam("rain_strength_min", "Rain strength min", params.RainStrengthMin),
				floatParam("rain_strength_max", "Rain strength max", params.RainStrengthMax),
			},
		},
		{
			Name: "Wind",
			Params: []core.Parameter{
				floatParam("wind_noise_scale", "Wind noise scale", params.WindNoiseScale),
				floatParam("wind_speed_scale", "Wind speed scale", params.WindSpeedScale),
				floatParam("wind_temporal_scale", "Wind temporal scale", params.WindTemporalScale),
			},
		},
		{
			Name: "Vegetation",
			Params: []core.Parameter{
				intParam("grass_neighbor_threshold", "Grass neighbor threshold", params.GrassNeighborThreshold),
				floatParam("grass_spread_chance", "Grass spread chance", params.GrassSpreadChance),
				intParam("shrub_neighbor_threshold", "Shrub neighbor threshold", params.ShrubNeighborThreshold),
				floatParam("shrub_growth_chance", "Shrub growth chance", params.ShrubGrowthChance),
				intParam("tree_neighbor_threshold", "Tree neighbor threshold", params.TreeNeighborThreshold),
				floatParam("tree_growth_chance", "Tree growth chance", params.TreeGrowthChance),
			},
		},
		{
			Name: "Volcano",
			Params: []core.Parameter{
				intParam("volcano_proto_max_regions", "Volcano proto max regions", params.VolcanoProtoMaxRegions),
				floatParam("volcano_proto_spawn_chance", "Volcano proto spawn chance", params.VolcanoProtoSpawnChance),
				floatParam("volcano_proto_tectonic_threshold", "Volcano proto tectonic threshold", params.VolcanoProtoTectonicThreshold),
				intParam("volcano_proto_radius_min", "Volcano proto radius min", params.VolcanoProtoRadiusMin),
				intParam("volcano_proto_radius_max", "Volcano proto radius max", params.VolcanoProtoRadiusMax),
				intParam("volcano_proto_ttl_min", "Volcano proto TTL min", params.VolcanoProtoTTLMin),
				intParam("volcano_proto_ttl_max", "Volcano proto TTL max", params.VolcanoProtoTTLMax),
				floatParam("volcano_proto_strength_min", "Volcano proto strength min", params.VolcanoProtoStrengthMin),
				floatParam("volcano_proto_strength_max", "Volcano proto strength max", params.VolcanoProtoStrengthMax),
				floatParam("volcano_uplift_chance_base", "Volcano uplift chance base", params.VolcanoUpliftChanceBase),
				floatParam("volcano_eruption_chance_base", "Volcano eruption chance base", params.VolcanoEruptionChanceBase),
			},
		},
	}
	return core.ParameterSnapshot{Groups: groups}
}

func intParam(key, label string, value int) core.Parameter {
	return core.Parameter{
		Key:   key,
		Label: label,
		Type:  core.ParamTypeInt,
		Value: strconv.Itoa(value),
	}
}

func int64Param(key, label string, value int64) core.Parameter {
	return core.Parameter{
		Key:   key,
		Label: label,
		Type:  core.ParamTypeInt,
		Value: strconv.FormatInt(value, 10),
	}
}

func floatParam(key, label string, value float64) core.Parameter {
	return core.Parameter{
		Key:   key,
		Label: label,
		Type:  core.ParamTypeFloat,
		Value: strconv.FormatFloat(value, 'f', -1, 64),
	}
}
