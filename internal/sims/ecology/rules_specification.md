# 🌋 Cellular Automata Ecology Simulation — Rules Specification

## 1. Overview

The ecology simulation is a deterministic 2‑D cellular automaton that couples vegetation succession, wildfire, volcanic activity, lava flow, and regional rainfall. Each tick blends stochastic choices (driven by the world seed) with tunable parameters so that the landscape cycles through growth, disturbance, and recovery.

---

## 2. World Representation

### 2.1 Discrete layers

| Layer        | States (enum order)                     | Notes |
| ------------ | --------------------------------------- | ----- |
| **Ground**   | `Dirt`, `Rock`, `Mountain`, `Lava`      | Exactly one per tile. `Lava` cells also store lava field data. |
| **Vegetation** | `None`, `Grass`, `Shrub`, `Tree`      | Updated after fire/lava processing each tick. |

### 2.2 Per-tile auxiliary fields

| Field                | Type / Range             | Purpose |
| -------------------- | ------------------------ | ------- |
| `lavaHeight`         | uint8 (0–7)              | Flow thickness. Zero when tile is not lava. |
| `lavaTemp`           | float32 (0–1)            | Cooling drives solidification. |
| `lavaDir`            | int8 (−1=no flow, 0–7 direction) | Heading index for advancing tips. |
| `lavaTip`            | bool                     | Marks active flow fronts. |
| `lavaForce`          | bool                     | Forces overflow advancement when height ≥4. |
| `lavaChannel`        | float32 (≥0)             | Memory of prior flow that biases routing. |
| `lavaElevation`      | int16                    | Pseudo-elevation raster constructed per eruption. |
| `burnTTL`            | uint8 (ticks remaining)  | Non-zero values denote burning vegetation. |
| `rainMask`           | float32 [0,1]            | Influence map rasterized from active rain regions. |
| `volcanoMask`        | float32 [0,1]            | Influence map rasterized from proto-volcano regions. |

### 2.3 Regional & global data

* `rainRegions`: active drifting rain clouds.
* `volcanoRegions`: active proto-volcano uplift regions.
* `expiredVolcanoProtos`: recently expired uplift regions awaiting eruption checks.
* `lavaVents`: active vents injecting lava into flow fields.
* `tectonic`: static 0–1 raster used to bias volcano spawning.
* Deterministic wind phase drives a curl-noise wind field shared by rain motion and HUD overlays.

---

## 3. Simulation Step

Each call to `Step()` performs the phases below in order. All random draws come from the seed-stable RNG stored on the world.

| Order | Phase | Key effects |
| ----- | ----- | ----------- |
| 1 | **Rain mask update** | Advance rain regions, merge overlaps, rasterize masks, and run morphology cleanup. |
| 2 | **Volcano mask update** | Advance proto-volcano regions, rasterize uplift mask, collect expired regions. |
| 3 | **Uplift** | Convert `Rock`→`Mountain` using volcano mask weights. |
| 4 | **Eruptions** | Expired proto regions may erupt, seeding lava cores/vents and rebuilding lava elevation. |
| 5 | **Lava dynamics** | Vent injection, flow advancement, pooling, cooling, and channel decay/growth. |
| 6 | **Fire** | Update burning TTLs, extinction, spread, and lava-ignited fires. |
| 7 | **Vegetation succession** | Apply growth transitions for non-burning tiles using neighbor counts. |
| 8 | **Region spawning** | Attempt to spawn new rain and proto-volcano regions. |
| 9 | **Display/metrics** | Refresh cached render buffers and aggregate vegetation metrics. |

---

## 4. Regional Rain Events

### 4.1 Spawning & lifecycle

* At most `RainMaxRegions` storms can exist (default 4). Up to two spawn attempts occur each tick, limited by remaining capacity.【F:internal/sims/ecology/ecology.go†L1087-L1112】【F:internal/sims/ecology/config.go†L80-L98】
* Each attempt rolls `RainSpawnChance` (default 0.22). Coverage above 15 % introduces a skip chance up to 90 % so storms thin out when the map is saturated.【F:internal/sims/ecology/ecology.go†L1113-L1135】
* Regions carry `ttl`, age, base strength, elliptical radii, perlin noise offsets, wind velocity, and a preset that shapes geometry.【F:internal/sims/ecology/ecology.go†L1141-L1494】
* Wind advection eases velocity toward the curl-noise wind vector with inertia 0.08 and caps step length at 0.8 tiles/tick. Nearby storms (<50 tiles) gently align velocities (cohesion blend 0.08).【F:internal/sims/ecology/ecology.go†L1204-L1256】
* Strength oscillates with a sine envelope whose swing is 10–20 % of the base value, giving natural ramp-up and decay.【F:internal/sims/ecology/ecology.go†L1268-L1283】

### 4.2 Presets

`makeRainRegion` selects among:

| Preset  | Traits |
| ------- | ------ |
| **Puffy** (55 % chance) | Radius sampled 16–40, circular, falloff 1.12–1.20. |
| **Stratus** (30 %) | Flattened band (`radiusY ≈ 0.6 × radiusX`), softer noise and falloff 1.08–1.16. |
| **Squall** (15 %) | Fast, elongated storm (`radiusX` up to 1.5× max, `radiusY` 10–16, TTL forced to 8–15 ticks). |

Base strength rolls between `max(0.5, RainStrengthMin)` and `RainStrengthMax` (defaults 0.5–1.0). Radii respect config limits (16–40 by default) and world size.【F:internal/sims/ecology/ecology.go†L1426-L1494】【F:internal/sims/ecology/config.go†L80-L98】

### 4.3 Overlap & morphology

* Storms whose overlap ratio exceeds 0.15 trigger blending: the larger cloud grows, gains +0.1 target strength, and both enter an 8-tick merge window while the smaller fades out.【F:internal/sims/ecology/ecology.go†L904-L940】
* Rasterization samples fBm noise (octaves=3) and radial distance. Noise values are thresholded with `smoothstep(threshold±0.1)` where threshold ∈ [0.42,0.52], guaranteeing solid cores (radial <0.45 → full opacity). Final mask value is `smoothstep(0,1,1−radial^falloff) × strength` with falloff ≈1.05–1.23.【F:internal/sims/ecology/ecology.go†L957-L1050】
* Morphological cleanup performs a 3px closing, 1px opening, then removes islands smaller than 25 cells (values >0.05).【F:internal/sims/ecology/ecology.go†L1222-L1287】

### 4.4 Coupling into simulation

* Lava cooling subtracts `ΔT = 0.02 + 0.03·edge + 0.08·rain + 0.02·thicknessSigmoid` (+0.02 if the cell is pooling).【F:internal/sims/ecology/ecology.go†L2772-L2811】
* Lava flow scoring penalizes rain via `score -= 0.5 × rain` before comparing against the flow threshold (0.9).【F:internal/sims/ecology/ecology.go†L2606-L2643】
* Fire spread and lava ignition chances are multiplied by `1 − FireRainSpreadDampen × rain` (clamped to [0,1]); default dampen is 0.75.【F:internal/sims/ecology/ecology.go†L2897-L2978】【F:internal/sims/ecology/config.go†L80-L98】
* Burning tiles extinguish with probability `FireRainExtinguishChance × rain` each tick (default 0.5).【F:internal/sims/ecology/ecology.go†L2930-L2957】

---

## 5. Volcano Proto Regions & Eruptions

### 5.1 Spawning

* At most `VolcanoProtoMaxRegions` uplift zones persist (default 6). Each tick rolls `VolcanoProtoSpawnChance` (default 0.02); success selects the noisiest high-tectonic tile among eight samples. Spawns require tectonic ≥ `VolcanoProtoTectonicThreshold` (default 0.6).【F:internal/sims/ecology/ecology.go†L2219-L2272】【F:internal/sims/ecology/config.go†L100-L118】
* Radius samples `VolcanoProtoRadiusMin`–`Max` (default 10–22). `ttl` samples `VolcanoProtoTTLMin`–`Max` (default 10–25). Strength samples `VolcanoProtoStrengthMin`–`Max` and is clamped ≤1.【F:internal/sims/ecology/ecology.go†L2266-L2307】【F:internal/sims/ecology/config.go†L100-L118】

### 5.2 Mask & uplift

* Rasterization writes a linear falloff disc into the volcano mask each tick: `value = strength × (1 − distance/radius)` (clamped to [0,1]).【F:internal/sims/ecology/ecology.go†L1865-L1914】
* During uplift, every `Rock` cell rolls `VolcanoUpliftChanceBase × mask` (default base 2e-5) to convert to `Mountain`.【F:internal/sims/ecology/ecology.go†L1916-L1950】【F:internal/sims/ecology/config.go†L100-L118】

### 5.3 Eruptions

* When a proto region expires, it computes the mean mask value across its footprint. An eruption occurs if a random roll < `VolcanoEruptionChanceBase × mean`, default base 5e-5.【F:internal/sims/ecology/ecology.go†L1952-L2006】【F:internal/sims/ecology/config.go†L100-L118】
* Eruptions clear existing lava, rebuild elevation, and seed:
  * **Core** (`r < 0.35R`): lava cells with height 2–3, temperature 1.0, and queued as tips.
  * **Rim** (`0.35R–0.9R`): `Rock` becomes `Mountain`.
  * **Vents**: 1–3 vents pick random core cells, run 20–40 ticks, inject 1 unit of lava per tick, and set initial outflow headings along downslope neighbors.【F:internal/sims/ecology/ecology.go†L2008-L2254】

---

## 6. Lava Dynamics

* **Injection:** Each vent increases its tile’s lava height (capped at 7) and temperature to 1.0. Neighboring outflow cells inherit at least height 1, become lava, and are marked as tips. Vegetation and burning data on affected cells are cleared immediately.【F:internal/sims/ecology/ecology.go†L2336-L2408】
* **Tip advancement:** Tips attempt to move forward each tick. The movement chance is `lavaBaseSpeed × temp / (1 + lavaSpeedAlpha × height)` unless forced by overflow. Candidate destinations score `1.0·slope + 0.6·alignment + 0.8·channel − 0.5·rain − 2.0·uphill`. Moves proceed when the best score ≥0.9 (or ≥0 when forced). If the source column is tall enough (height ≥3) and a second candidate scores within 0.75, a split may spawn an extra branch (25 % chance).【F:internal/sims/ecology/ecology.go†L2568-L2709】
* **Pooling & overflow:** Failed tips thicken the trunk (up to height 7). They may fill adjacent low cells with stationary pools (`dir = -1`), which cool faster and can overflow later.【F:internal/sims/ecology/ecology.go†L2711-L2759】
* **Cooling & crusting:** Temperature falls by the formula in §4.4. When `temp ≤ 0.15`, thick flows shed one unit of height; once height reaches 1, cooled lava solidifies to `Rock`. Heat is clamped to ≤0.35 when crusting so flows can restart gently if reheated.【F:internal/sims/ecology/ecology.go†L2772-L2823】
* **Channel memory:** Tiles that advanced gain +0.15 channel weight (clamped ≤1). Global decay (0.5 % per tick) keeps flow paths semi-permanent without locking them forever.【F:internal/sims/ecology/ecology.go†L2825-L2844】
* **Tip detection:** After updates, tips are recomputed using temperature, height, and local lava connectivity (≤2 lava neighbors).【F:internal/sims/ecology/ecology.go†L2846-L2881】

---

## 7. Fire System

* Burning tiles count down `BurnTTL` (default 3). When the counter hits zero the vegetation becomes `None` and the display reverts to the ground layer.【F:internal/sims/ecology/ecology.go†L2890-L2968】【F:internal/sims/ecology/config.go†L72-L98】
* Spread attempts visit all Moore neighbors. Each vegetation tile not already burning rolls `FireSpreadChance` (default 0.25) scaled by the rain modifier described in §4.4. Successful ignitions enqueue TTL = `BurnTTL` (clamped ≤255).【F:internal/sims/ecology/ecology.go†L2968-L3017】
* Lava ignition checks vegetation adjacent to lava tiles and applies `FireLavaIgniteChance` (default 0.8) with the same rain damping. Ignitions write TTL directly into `burnNext`.【F:internal/sims/ecology/ecology.go†L3019-L3073】【F:internal/sims/ecology/config.go†L72-L98】

---

## 8. Vegetation Succession

Vegetation updates after fire, using cached Moore neighbor counts for grass and shrubs.

| Transition | Condition | Probability | Default neighbors |
| ---------- | --------- | ----------- | ----------------- |
| Dirt → Grass | Tile is `Dirt`, at least `GrassNeighborThreshold` grass neighbors, and random < `GrassSpreadChance`. | Configurable (defaults: threshold 1, chance 0.25). |
| Grass → Shrub | At least `ShrubNeighborThreshold` grass neighbors and random < `ShrubGrowthChance`. | Defaults: threshold 3, chance 0.04. |
| Shrub → Tree | At least `TreeNeighborThreshold` shrub neighbors and random < `TreeGrowthChance`. | Defaults: threshold 3, chance 0.02. |

Burning tiles skip succession until extinguished. Metrics update after writing `vegNext` and buffers swap.【F:internal/sims/ecology/ecology.go†L818-L876】【F:internal/sims/ecology/config.go†L58-L79】

---

## 9. Initialization & Tunables

`DefaultConfig()` creates a 256×256 world with deterministic seed 1337 and the parameter pack in `config.go`. Highlights include:

* Terrain: `RockChance` 5 %, grass patch count 12 with radii 2–5 and density 0.6.【F:internal/sims/ecology/config.go†L64-L88】
* Lava vent lifetime defaults to 20–40 ticks (`LavaLifeMin/Max`), and spread floor `LavaSpreadMaskFloor` 0.2 (currently unused but exposed for tuning).【F:internal/sims/ecology/config.go†L64-L83】
* Wind: `WindNoiseScale` 0.01, `WindSpeedScale` 0.6, `WindTemporalScale` 0.05.【F:internal/sims/ecology/config.go†L80-L98】
* All parameters are adjustable at runtime via the HUD parameter snapshot plumbing, and `FromMap` supports overriding values from CLI-style maps.【F:internal/sims/ecology/config.go†L120-L323】

---

## 10. Long-term Behaviour

The interplay of systems drives a repeating ecological loop:

1. Grass spreads and matures into shrubs and trees.
2. Proto-volcano regions uplift mountains and occasionally erupt.
3. Lava rivers carve paths, burn vegetation, and cool into new rock, influenced by rain.
4. Fires ignite from lava and propagate across vegetation, with rain suppressing spread and extinguishing edges.
5. Fresh rock/dirt clears the way for vegetation succession to restart, completing the cycle.

Deterministic seeding plus telemetry collectors (vegetation and environmental metrics) support regression testing and tuning of these dynamics.【F:internal/sims/ecology/ecology.go†L24-L118】【F:internal/sims/ecology/ecology.go†L3088-L3242】
