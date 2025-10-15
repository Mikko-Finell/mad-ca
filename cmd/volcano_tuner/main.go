package main

import (
	"flag"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"mad-ca/internal/sims/ecology"
)

type kvList []string

func (l *kvList) String() string {
	return strings.Join(*l, ",")
}

func (l *kvList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func main() {
	steps := flag.Int("steps", 600, "number of ticks to simulate per candidate")
	passes := flag.Int("passes", 3, "coordinate-descent passes to execute")
	workers := flag.Int("workers", runtime.NumCPU(), "parallel candidate evaluations")
	width := flag.Int("width", 192, "map width for tuning runs")
	height := flag.Int("height", 192, "map height for tuning runs")
	seed := flag.Int64("seed", 1337, "seed used for deterministic simulations")
	manualOnly := flag.Bool("manual", false, "skip sweeping and only evaluate provided overrides")
	var overrides kvList
	flag.Var(&overrides, "set", "parameter override in key=value form (repeatable)")
	flag.Parse()

	cfg := ecology.DefaultConfig()
	cfg.Width = *width
	cfg.Height = *height
	cfg.Seed = *seed
	cfg.Params.GrassPatchCount = 0
	cfg.Params.RainSpawnChance = 0
	cfg.Params.RainMaxRegions = 0
	cfg.Params.VolcanoProtoSpawnChance = 0
	cfg.Params.VolcanoEruptionChanceBase = 0
	cfg.Params.FireSpreadChance = 0
	cfg.Params.FireLavaIgniteChance = 0

	for _, kv := range overrides {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		applyOverride(&cfg.Params, parts[0], parts[1])
	}

	baseline := ecology.VolcanoFlowResult(cfg, *steps)
	diameter := ecology.VolcanoDiameter(cfg.Params)
	ratio := 0.0
	if diameter > 0 {
		ratio = baseline.MaxDistance / diameter
	}

	fmt.Printf("Baseline: max distance %.2f (diameter %.2f, ratio %.2f), last lava step %d/%d, peak tiles %d, max step %d, initial tips %d\n",
		baseline.MaxDistance, diameter, ratio, baseline.LastActiveStep, baseline.StepsSimulated, baseline.PeakActiveTiles, baseline.MaxDistanceStep, baseline.InitialTipCount)

	if *manualOnly {
		fmt.Println("Manual evaluation requested; skipping sweep.")
		printParams(cfg.Params)
		return
	}

	params, result, trace := ecology.VolcanoParameterSweep(cfg, *steps, *passes, *workers)
	tunedDiameter := ecology.VolcanoDiameter(params)
	tunedRatio := 0.0
	if tunedDiameter > 0 {
		tunedRatio = result.MaxDistance / tunedDiameter
	}

	fmt.Printf("\nBest found: max distance %.2f (diameter %.2f, ratio %.2f), last lava step %d/%d, peak tiles %d, max step %d, initial tips %d\n",
		result.MaxDistance, tunedDiameter, tunedRatio, result.LastActiveStep, result.StepsSimulated, result.PeakActiveTiles, result.MaxDistanceStep, result.InitialTipCount)
	printParams(params)

	if len(trace) > 1 {
		fmt.Println("\nImprovements:")
		for _, rec := range trace[1:] {
			r := 0.0
			d := ecology.VolcanoDiameter(rec.Params)
			if d > 0 {
				r = rec.Result.MaxDistance / d
			}
			fmt.Printf("  pass %d: %s=%s -> maxDist=%.2f, last lava step %d (ratio %.2f)\n",
				rec.Pass, rec.Parameter, rec.Value, rec.Result.MaxDistance, rec.Result.LastActiveStep, r)
		}
	}
}

func applyOverride(params *ecology.Params, key, value string) {
	switch key {
	case "lava_flux_ref":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaFluxRef = v
		}
	case "lava_cool_base":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaCoolBase = v
		}
	case "lava_cool_edge":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaCoolEdge = v
		}
	case "lava_cool_thick":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaCoolThick = v
		}
	case "lava_cool_flux":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaCoolFlux = v
		}
	case "lava_slope_scale_min":
		if v, err := strconv.Atoi(value); err == nil {
			params.LavaSlopeScaleMin = v
		}
	case "lava_slope_scale_max":
		if v, err := strconv.Atoi(value); err == nil {
			params.LavaSlopeScaleMax = v
		}
	case "lava_flow_threshold":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaFlowThreshold = v
		}
	case "lava_slope_weight":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaSlopeWeight = v
		}
	case "lava_align_weight":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaAlignWeight = v
		}
	case "lava_channel_weight":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaChannelWeight = v
		}
	case "lava_rain_weight":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaRainWeight = v
		}
	case "lava_wall_weight":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaWallWeight = v
		}
	case "lava_reservoir_gain":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaReservoirGain = v
		}
	case "lava_reservoir_head":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaReservoirHead = v
		}
	case "lava_reservoir_min":
		if v, err := strconv.Atoi(value); err == nil {
			params.LavaReservoirMin = v
			if params.LavaReservoirMax < v {
				params.LavaReservoirMax = v
			}
		}
	case "lava_reservoir_max":
		if v, err := strconv.Atoi(value); err == nil {
			if v < params.LavaReservoirMin {
				v = params.LavaReservoirMin
			}
			params.LavaReservoirMax = v
		}
	case "lava_phase_threshold":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaPhaseThreshold = v
		}
	case "lava_phase_hysteresis":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.LavaPhaseHysteresis = v
		}
	case "volcano_proto_strength_min":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.VolcanoProtoStrengthMin = v
		}
	case "volcano_proto_strength_max":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			params.VolcanoProtoStrengthMax = v
		}
	}
}

func printParams(params ecology.Params) {
	fmt.Println("Parameters:")
	fmt.Printf("  lava_flux_ref=%.3f\n", params.LavaFluxRef)
	fmt.Printf("  lava_cool_base=%.3f\n", params.LavaCoolBase)
	fmt.Printf("  lava_cool_edge=%.3f\n", params.LavaCoolEdge)
	fmt.Printf("  lava_cool_thick=%.3f\n", params.LavaCoolThick)
	fmt.Printf("  lava_cool_flux=%.3f\n", params.LavaCoolFlux)
	fmt.Printf("  lava_slope_scale_min=%d\n", params.LavaSlopeScaleMin)
	fmt.Printf("  lava_slope_scale_max=%d\n", params.LavaSlopeScaleMax)
	fmt.Printf("  lava_flow_threshold=%.3f\n", params.LavaFlowThreshold)
	fmt.Printf("  lava_slope_weight=%.3f\n", params.LavaSlopeWeight)
	fmt.Printf("  lava_align_weight=%.3f\n", params.LavaAlignWeight)
	fmt.Printf("  lava_channel_weight=%.3f\n", params.LavaChannelWeight)
	fmt.Printf("  lava_rain_weight=%.3f\n", params.LavaRainWeight)
	fmt.Printf("  lava_wall_weight=%.3f\n", params.LavaWallWeight)
	fmt.Printf("  lava_reservoir_gain=%.3f\n", params.LavaReservoirGain)
	fmt.Printf("  lava_reservoir_head=%.3f\n", params.LavaReservoirHead)
	fmt.Printf("  lava_reservoir_min=%d\n", params.LavaReservoirMin)
	fmt.Printf("  lava_reservoir_max=%d\n", params.LavaReservoirMax)
	fmt.Printf("  lava_phase_threshold=%.3f\n", params.LavaPhaseThreshold)
	fmt.Printf("  lava_phase_hysteresis=%.3f\n", params.LavaPhaseHysteresis)
	fmt.Printf("  volcano_proto_strength_min=%.3f\n", params.VolcanoProtoStrengthMin)
	fmt.Printf("  volcano_proto_strength_max=%.3f\n", params.VolcanoProtoStrengthMax)
}
