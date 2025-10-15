package main

import (
	"flag"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"mad-ca/internal/sims/ecology"
)

type paramSet struct {
	reservoirMin  int
	reservoirMax  int
	reservoirGain float64
	reservoirHead float64
	coolBase      float64
	coolThick     float64
	coolFlux      float64
	spreadChance  float64
	fluxRef       float64
}

func (p paramSet) String() string {
	return fmt.Sprintf("min=%d max=%d gain=%.2f head=%.2f coolBase=%.3f coolThick=%.3f coolFlux=%.3f spread=%.3f fluxRef=%.2f",
		p.reservoirMin, p.reservoirMax, p.reservoirGain, p.reservoirHead, p.coolBase, p.coolThick, p.coolFlux, p.spreadChance, p.fluxRef)
}

type scenarioResult struct {
	params          paramSet
	maxDistance     float64
	stepReached     int
	lavaTilePeak    int
	maxHeight       uint8
	maxTemp         float64
	tipPeak         int
	minElevation    int16
	maxElevation    int16
	initialTipCount int
	centerElevation int16
	eastElevation   int16
}

func main() {
	steps := flag.Int("steps", 240, "ticks to simulate per scenario")
	workers := flag.Int("workers", runtime.NumCPU(), "number of worker goroutines")
	flag.Parse()

	baseCfg := ecology.DefaultConfig()
	baseCfg.Width = 160
	baseCfg.Height = 160
	baseCfg.Params.GrassPatchCount = 0
	baseCfg.Params.RockChance = 0.0
	baseCfg.Params.RainSpawnChance = 0
	baseCfg.Params.RainMaxRegions = 0
	baseCfg.Params.VolcanoProtoSpawnChance = 0

	headOptions := []float64{5.5, 6.5, 7.5}
	gainOptions := []float64{1.5, 2.0, 2.5}
	reservoirOptions := []struct{ min, max int }{
		{min: 320, max: 420},
		{min: 480, max: 620},
		{min: 720, max: 900},
	}
	coolingOptions := []struct {
		base  float64
		thick float64
		flux  float64
	}{
		{base: 0.008, thick: 0.010, flux: 0.010},
		{base: 0.006, thick: 0.008, flux: 0.008},
		{base: 0.004, thick: 0.006, flux: 0.006},
	}
	spreadOptions := []float64{0.14, 0.18, 0.22}
	fluxRefOptions := []float64{3.5, 4.5, 5.5}

	var sets []paramSet
	for _, head := range headOptions {
		for _, gain := range gainOptions {
			for _, res := range reservoirOptions {
				for _, cool := range coolingOptions {
					for _, spread := range spreadOptions {
						for _, flux := range fluxRefOptions {
							sets = append(sets, paramSet{
								reservoirMin:  res.min,
								reservoirMax:  res.max,
								reservoirGain: gain,
								reservoirHead: head,
								coolBase:      cool.base,
								coolThick:     cool.thick,
								coolFlux:      cool.flux,
								spreadChance:  spread,
								fluxRef:       flux,
							})
						}
					}
				}
			}
		}
	}

	fmt.Printf("Sweeping %d parameter sets (%d workers, %d steps)\n", len(sets), *workers, *steps)

	jobs := make(chan paramSet)
	results := make(chan scenarioResult)
	var wg sync.WaitGroup

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for params := range jobs {
				res := runScenario(baseCfg, params, *steps)
				results <- res
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, params := range sets {
			jobs <- params
		}
		close(jobs)
	}()

	start := time.Now()
	var all []scenarioResult
	best := scenarioResult{}
	threshold := volcanoDiameter(baseCfg)
	for res := range results {
		all = append(all, res)
		if res.maxDistance > best.maxDistance {
			best = res
		}
		if res.maxDistance >= threshold {
			fmt.Printf("Candidate reached %.2f (threshold %.2f) at step %d with %s\n",
				res.maxDistance, threshold, res.stepReached, res.params)
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].maxDistance > all[j].maxDistance })
	elapsed := time.Since(start)

	fmt.Printf("\nTop 5 results (elapsed %s):\n", elapsed.Round(time.Millisecond))
	for i := 0; i < len(all) && i < 5; i++ {
		res := all[i]
		fmt.Printf("%2d) dist=%.2f step=%d lavaTiles=%d tips=%d/%d maxHeight=%d maxTemp=%.2f elev[%d,%d] center=%d east=%d params=%s\n",
			i+1, res.maxDistance, res.stepReached, res.lavaTilePeak, res.tipPeak, res.initialTipCount, res.maxHeight, res.maxTemp, res.minElevation, res.maxElevation, res.centerElevation, res.eastElevation, res.params)
	}

	fmt.Printf("\nBest overall: dist=%.2f step=%d lavaTiles=%d tips=%d/%d maxHeight=%d maxTemp=%.2f elev[%d,%d] center=%d east=%d params=%s\n",
		best.maxDistance, best.stepReached, best.lavaTilePeak, best.tipPeak, best.initialTipCount, best.maxHeight, best.maxTemp, best.minElevation, best.maxElevation, best.centerElevation, best.eastElevation, best.params)
}

func runScenario(base ecology.Config, params paramSet, steps int) scenarioResult {
	cfg := base
	cfg.Params.LavaReservoirMin = params.reservoirMin
	cfg.Params.LavaReservoirMax = params.reservoirMax
	cfg.Params.LavaReservoirGain = params.reservoirGain
	cfg.Params.LavaReservoirHead = params.reservoirHead
	cfg.Params.LavaCoolBase = params.coolBase
	cfg.Params.LavaCoolThick = params.coolThick
	cfg.Params.LavaCoolFlux = params.coolFlux
	cfg.Params.LavaCoolEdge = base.Params.LavaCoolEdge
	cfg.Params.LavaCoolRain = base.Params.LavaCoolRain
	cfg.Params.LavaSpreadChance = params.spreadChance
	cfg.Params.LavaFluxRef = params.fluxRef

	world := ecology.NewWithConfig(cfg)
	world.Reset(1337)

	ground := world.Ground()
	for i := range ground {
		ground[i] = ecology.GroundRock
	}
	veg := world.Vegetation()
	for i := range veg {
		veg[i] = ecology.VegetationNone
	}

	centerX := cfg.Width / 2
	centerY := cfg.Height / 2
	world.SpawnVolcanoAt(centerX, centerY)

	center := struct{ x, y float64 }{
		x: float64(centerX) + 0.5,
		y: float64(centerY) + 0.5,
	}

	size := world.Size()
	width := size.W

	target := volcanoDiameter(cfg)
	cutoff := target * 1.5

	var maxDist float64
	var stepReached int
	var peakLava int
	var maxHeight uint8
	var maxTemp float64
	var peakTips int

	elevation := world.ElevationField()
	var minElev, maxElev int16
	if len(elevation) > 0 {
		minElev = elevation[0]
		maxElev = elevation[0]
		for _, v := range elevation {
			if v < minElev {
				minElev = v
			}
			if v > maxElev {
				maxElev = v
			}
		}
	}

	centerIdx := centerY*width + centerX
	eastIdx := centerIdx
	if centerX+1 < width {
		eastIdx = centerY*width + (centerX + 1)
	}
	centerElev := int16(0)
	eastElev := int16(0)
	if centerIdx >= 0 && centerIdx < len(elevation) {
		centerElev = elevation[centerIdx]
	}
	if eastIdx >= 0 && eastIdx < len(elevation) {
		eastElev = elevation[eastIdx]
	}

	initialTips := countTrue(world.LavaTipField())

	for step := 0; step < steps; step++ {
		world.Step()

		ground := world.Ground()
		heights := world.LavaHeightField()
		temps := world.LavaTemperatureField()
		tips := world.LavaTipField()
		lavaTiles := 0
		for idx, tile := range ground {
			if tile != ecology.GroundLava {
				continue
			}
			lavaTiles++
			x := float64(idx%width) + 0.5
			y := float64(idx/width) + 0.5
			dist := math.Hypot(x-center.x, y-center.y)
			if dist > maxDist {
				maxDist = dist
				stepReached = step + 1
			}
			if idx < len(heights) {
				h := heights[idx]
				if h > maxHeight {
					maxHeight = h
				}
			}
			if idx < len(temps) {
				t := float64(temps[idx])
				if t > maxTemp {
					maxTemp = t
				}
			}
		}
		if lavaTiles > peakLava {
			peakLava = lavaTiles
		}
		if tipCount := countTrue(tips); tipCount > peakTips {
			peakTips = tipCount
		}
		if maxDist >= cutoff {
			break
		}
	}

	return scenarioResult{
		params:          params,
		maxDistance:     maxDist,
		stepReached:     stepReached,
		lavaTilePeak:    peakLava,
		maxHeight:       maxHeight,
		maxTemp:         maxTemp,
		tipPeak:         peakTips,
		minElevation:    minElev,
		maxElevation:    maxElev,
		initialTipCount: initialTips,
		centerElevation: centerElev,
		eastElevation:   eastElev,
	}
}

func countTrue(values []bool) int {
	total := 0
	for _, v := range values {
		if v {
			total++
		}
	}
	return total
}

func volcanoDiameter(cfg ecology.Config) float64 {
	radius := float64(cfg.Params.VolcanoProtoRadiusMin)
	if cfg.Params.VolcanoProtoRadiusMax > cfg.Params.VolcanoProtoRadiusMin {
		radius = float64(cfg.Params.VolcanoProtoRadiusMin + (cfg.Params.VolcanoProtoRadiusMax-cfg.Params.VolcanoProtoRadiusMin)/2)
	}
	return radius * 2
}
