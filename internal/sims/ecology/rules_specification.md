# 🌋 Cellular Automata Ecology Simulation

### Version 1.1 — Regional Events Edition

## 1. Overview

This 2-D cellular automaton models the evolution of an **ecological-geological landscape**: vegetation growth, forest fires, volcanic activity, and rainfall.
It is not meant as a pure mathematical automaton but as a simulation-style system whose rules yield plausible terrain cycles: grasslands spreading, forests forming, volcanoes erupting, lava cooling, and regrowth after disturbance.

---

## 2. World Structure

| Layer               | States               | Notes                             |          |       |                        |
| ------------------- | -------------------- | --------------------------------- | -------- | ----- | ---------------------- |
| **Ground**          | `Dirt                | Rock                              | Mountain | Lava` | Mutually exclusive.    |
| **Vegetation**      | `None                | Grass                             | Shrub    | Tree` | Independent of ground. |
| **Transient flags** | `Burning`, `RainWet` | Short-lived, derived from events. |          |       |                        |

**Neighborhood:** Moore (8 neighbors)
**Ticking:** Deterministic; use seeded PRNG for stochastic rules.
**Simulation order:**

1. Build region masks (rain & volcano)
2. Tectonics / volcano uplift & eruptions
3. Lava spread & cooling
4. Fire ignition & propagation
5. Vegetation succession & spread
6. Cleanup & region spawning

---

## 3. Environmental Fields

Optional continuous maps influencing probabilities:

| Field               | Range | Role                                     |
| ------------------- | ----- | ---------------------------------------- |
| `tectonic_map[x,y]` | 0–1   | drives mountain uplift & volcano seeding |
| `RainMask[x,y]`     | 0–1   | produced by regional rain events         |
| `VolcanoMask[x,y]`  | 0–1   | produced by proto-volcano regions        |

---

## 4. Region Events

### 4.1 Concept

A **RegionEvent** defines an area-scale temporary influence.

```
kind: "Rain" | "VolcanoProto"
cx, cy : float
r       : float
falloff : "gaussian" | "linear" | "flat"
strength: float  (0–1)
ttl     : int    // ticks remaining
noiseSeed: int   // for irregular edges
```

Each tick, events rasterize into their masks; when `ttl==0` they expire.

---

### 4.2 Rain Regions

* Spawn 0–2 new regions per tick (cap ≈4 active). Spawn is suppressed when existing clouds already cover >15% of the map.
* Each region spans radius 16–40 tiles (anisotropic variants stretch axes) and lasts 12–30 ticks by default. Squall presets use 8–15 ticks.
* Mask geometry combines noise-thresholded blobs with smooth radial falloff: `R(x,y) = clamp(step(τ, fbm((x,y))) * smoothstep(0,1,1-(d/R)^p) * strength, 0,1)` with `p≈1.3–1.5`, `τ≈0.35–0.55`, and per-region noise seeds. Small specks are removed with a 1px morphological closing pass.
* Regions drift each tick using a low-frequency wind field plus light jitter; strength eases in/out (~±15%). Overlapping clouds (>20% shared area) are merged by max-blending and the larger cloud absorbs the smaller (strength +0.1, capped at 1.0).
* Presets provide variety: puffy (round), stratus (flattened band, stretched noise), and squall (elongated major axis, faster drift).

**Effects (sample R = RainMask[x,y]):**

| Rule               | Multiplier         |
| ------------------ | ------------------ |
| Lava cooling bonus | `+8 × R` per tick  |
| Ignite/spread prob | `× (1 − 0.75 × R)` |
| Extinguish chance  | `0.5 × R`          |

Rain thus cools lava and damps fires smoothly across its gradient.

---

### 4.3 Volcano Proto Regions

Volcano formation is a **two-phase process**.

#### Phase A – Proto (uplift)

* Spawn region where tectonic_map is high.
* Radius 10–22, lifetime 10–25 ticks, nearly flat falloff with slight jitter.
* Each tick: if `ground==Rock` and random < `2×P_uplift_base×V`, convert to `Mountain`.
* Uplift probability may peak near the rim to form a caldera.

#### Phase B – Eruption

When proto expires:

* Compute mean mask value V̄.
* With chance `P_erupt_base×V̄`, erupt:

  * **Core** (`r < 0.35R`): `ground = Lava`, random lava_life.
  * **Rim** (`0.35R–0.9R`): convert Rock→Mountain.
  * **Specks:** small random lava spots on rim.
* If not erupted → region vanishes quietly.

---

## 5. Ground-Layer Rules

| Step             | Condition                                               | Result                |
| ---------------- | ------------------------------------------------------- | --------------------- |
| **Uplift**       | Rock + tectonic chance                                  | → Mountain            |
| **Eruption**     | Mountain + erupt chance                                 | → Lava                |
| **Lava Spread**  | Lava adjacent to Dirt/Rock, chance `P_lava_spread_edge` | neighbor → Lava       |
| **Lava Cooling** | each tick reduce life; extra cool bonus from rain       | Lava→Rock when life≤0 |

Defaults

```
P_uplift_base = 0.00002
P_erupt_base  = 0.00005
lava_life = 15–40 ticks
P_lava_spread_edge = 0.08
```

---

## 6. Fire Rules

| Stage                  | Logic                                                   |
| ---------------------- | ------------------------------------------------------- |
| **Ignition from Lava** | Vegetation near Lava ignites with `0.8 × (1−0.75R)`     |
| **Fire Spread**        | Burning neighbor ignites others with `0.25 × (1−0.75R)` |
| **Burn-Down**          | After 3 ticks, vegetation→None                          |
| **Rain Extinguish**    | `rand()<0.5R` → clear Burning                           |

---

## 7. Vegetation Rules

Executed top-down so a tile only advances one stage per tick.

| Transition  | Condition          | Probability |
| ----------- | ------------------ | ----------- |
| Shrub→Tree  | ≥3 Shrub neighbors | 0.02        |
| Grass→Shrub | ≥3 Grass neighbors | 0.04        |
| Dirt→Grass  | any Grass neighbor | 0.25        |

Fire or lava remove vegetation; there’s no passive withering.

---

## 8. Tick Sequence (Pseudo)

```pseudo
tick():
  RainMask.clear(); VolcanoMask.clear()

  // 1. Rasterize regions
  for e in regions:
    mask = (e.kind==Rain)?RainMask:VolcanoMask
    rasterizeRoundish(e, mask)
    e.ttl -= 1

  // 2. Volcano proto uplift & eruptions
  applyUplift(VolcanoMask)
  eruptExpiredProtos(VolcanoMask)

  // 3. Lava spread & cooling (uses RainMask)
  updateLava()

  // 4. Fire (uses RainMask)
  updateFire()

  // 5. Vegetation
  updateVegetation()

  // 6. Cleanup & spawn new regions
  removeExpiredRegions()
  maybeSpawnRain()
  maybeSpawnVolcanoProto()
```

---

## 9. Initialization

* Start ground as **Dirt**, sprinkle **Rock** clusters.
* Generate static `tectonic_map` with noisy gradients or ridges.
* Seed a few **Grass** patches.
* Begin with no active regions.

---

## 10. Implementation Notes

* Use **double buffering** for both layers.
* Store per-cell integers for lava life & burn TTL.
* Evaluate probabilities in random or shuffled order to reduce bias.
* Keep region count small; rasterization is cheap when few regions exist.
* For visuals, color by `ground` then overlay vegetation and burning glow.

---

### Summary

This version forms a closed ecological loop:

1. **Grasslands** expand → **Shrubs** → **Forests**.
2. **Volcano protoregions** uplift mountains and occasionally **erupt**.
3. **Lava** spreads, burns vegetation, **cools to rock**, restoring substrate.
4. **Rain regions** drift across the map, damping fires and hastening cooling.
5. Over many ticks, the map cycles through **growth, destruction, and renewal**, generating an emergent, believable terrain ecology.
