# Cellular-Ecology CA — Implementation Roadmap (5 Phases)

## Phase 1 — Model & Config Wiring

**Status:** Complete — deterministic world scaffolding in place and rain/volcano mask overlays renderable via debug toggles (`1` rain, `2` volcano). Ready to begin vegetation dynamics work.

**Goal:** Represent the world state and parameters, no behavior yet.

**Scope**

* Define two layers: **Ground** (`Dirt|Rock|Mountain|Lava`) and **Vegetation** (`None|Grass|Shrub|Tree`).
* Add per-cell auxiliaries: `lava_life:int`, `burn_ttl:int` (0 = inactive).
* Introduce **region masks** (float [0..1]): `RainMask`, `VolcanoMask` (double-buffered).
* Load a static `tectonic_map` ([0..1]) and a deterministic PRNG (seeded).
* Centralize tunables (thresholds/probabilities/lifetimes) in a params object.

**Tests & Tools**

* Determinism: same seed → identical layers after N ticks (no rules yet).

**Exit Criteria**

* World can be initialized (Dirt + Rock sprinkles + Grass seeds + tectonic_map).
* Params adjustable at runtime; masks exist and are rendered as overlays.

---

## Phase 2 — Vegetation Succession (No Fire Yet)

**Status:** Complete — vegetation succession plus telemetry for growth curve & patchiness metrics in place.

**Goal:** Get lifelike grass→shrub→tree growth and dirt→grass colonization.

**Scope**

* Implement neighbor counting (Moore 8) via engine’s stencil ops.
* Rules (single-step per tick):
  `Dirt→Grass` (adjacent Grass), `Grass→Shrub` (≥T_shrub Grass), `Shrub→Tree` (≥T_tree Shrub).
* Initialization helpers to place several grass patches.
* Keep double buffering for vegetation; no burning interactions yet.

**Tests & Telemetry**

* Growth curve sanity: total vegetation area increases from seeds.
* Patchiness: cluster size distribution not degenerate (no checkerboard).
* Deterministic replay for 1k ticks with fixed seed.

**Notes:**

* Implemented helper-based neighbor counting and per-tick succession updates with unit tests covering spread and stage advancement.
* Added vegetation telemetry (per-stage counts and cluster histogram) with deterministic regression tests verifying growth curve and patchiness.

**Exit Criteria**

* Stable spread/succession dynamics visible; parameters tune expected rates.

---

## Phase 3 — Fire System

**Status:** Complete — burning lifecycle, lava ignition, rain modulation, and lava cooling/spread now covered by tests with fire hooks in place.

**Goal:** Add Burning flag lifecycle and local fire spread (lava not required yet).

**Scope**

* `Burning` flag with `burn_ttl` countdown; on expiry → vegetation `None`.
* Fire spread: burning neighbors ignite vegetation with probability `P_fire_spread`.
* Manual triggers (dev tool) to ignite tiles for testing.
* Prepare hooks for **external multipliers** (to be driven by RainMask later).

**Tests & Telemetry**

* Fire front propagation speed within configured envelope.
* Burn-out leaves predictable clearings; no phantom fires (no negative TTLs).
* Determinism under mixed burning/non-burning neighborhoods.

**Notes:**

* Manual ignition debug control is available via the app and fire-related parameters are configurable. The burn TTL countdown now clears vegetation, spreads to neighbors using tunables, and is covered by unit tests. Lava-adjacent tiles can ignite and rain both dampens and extinguishes fires. Lava tiles now cool to rock, spread to nearby dirt/rock, clear vegetation, and immediately feed the ignition logic, enabling mask-driven modulation once region spawners arrive. Next phase: begin tectonic uplift and proto-volcano region plumbing.

**Exit Criteria**

* Fire behaves plausibly and stops without infinite oscillation.
* Multipliers are parameterized (but not yet wired to rain).

---

## Phase 4 — Tectonics, Volcano Proto, Lava

**Status:** Complete — proto-volcano region spawner, mask rasterization, uplift conversion, mask-weighted lava spread/cooling, and rain-aware lava regression coverage are locked in.

**Goal:** Geological engine: uplift, eruptions, lava spread/cool.

**Scope**

* **Region events framework:** list of active regions (center, radius, falloff, ttl, noise seed); per-tick rasterization to `VolcanoMask`. *(Volcano protoregion list implemented with linear falloff masks; expirations tracked for future eruption handling.)*
* **Proto-volcano uplift:** Rock→Mountain with chance scaled by `VolcanoMask * P_uplift_base`. *(Implemented — uplift now occurs after mask build.)*
* **Eruption on expiry:** chance `P_erupt_base * mean(VolcanoMask in region)`; write core lava, rim mountains, occasional lava specks.
* **Lava:** spread to Dirt/Rock with `P_lava_spread_edge`, per-cell `lava_life` countdown → Lava→Rock on zero.
* keep order: build masks → uplift/erupt → lava spread/cool → fire → vegetation. *(Tick now builds volcano masks before uplifting and the existing lava/fire/vegetation phases.)*

**Tests & Telemetry**

* Volcano lifecycle: proto appears → uplift rises → eruption sometimes occurs → lava cools to rock.
* Spread bounded (one-ring per tick) with double buffering—no mid-tick cascades.
* Long-run equilibrium check: repeated cycles don’t crash performance or blow up memory.

**Exit Criteria**

* Volcano protos form roundish uplift, eruptions produce credible cones/lava fields.
* Lava cools reliably; no stuck eternal lava without config asking for it.

**Notes:**

* Proto-volcano lifecycle now consumes proto expirations to trigger eruptions that seed lava cores, uplift rims, and clear vegetation while feeding the lava system. Lava spread and cooling now scale with the volcano mask so flows stay localized and cool between eruptions, rain-driven cooling bonuses feed into the lava step, and the long-run regression now exercises mixed rain/volcano scenarios. Next step: begin Phase 5 by introducing regional rain events that rasterize into the rain mask and drive the remaining multipliers.

---

## Phase 5 — Regional Rain Integration & Tuning

**Status:** In progress — rain regions now rasterize into the mask with gaussian falloff, fire/lava multipliers honor RainMask, and stochastic spawning plus expiry tests are in place.

**Goal:** Replace per-tile wetness with **Rain regions** and wire all multipliers.

**Scope**

* Rain regions rasterized to `RainMask` each tick (gaussian falloff, 8–24 radius, 8–20 ttl).
* Wire multipliers:

  * Ignite & spread: `× (1 − 0.75 * RainMask)`
  * Extinguish: probability `0.5 * RainMask`
  * Lava cooling bonus: `+round(8 * RainMask)` per tick
* Low-rate stochastic spawner for rain regions; cap concurrent count.

**Tests & Telemetry**

* Side-by-side: same seed w/ and w/o rain → cooler lava and smaller burn scars under rain tracks.
* “Stripe test”: inject a long rain band; confirm gradient effects (softer edges).
* Performance: mask rasterization scales with small region counts.

**Notes:**

* Added gaussian rain region rasterization, cap-aware spawning, and expiry coverage alongside regression updates for the new rain stripe behavior.
* Next step: tune spawn frequencies/strength ranges so mixed rain–volcano cycles regain pre-rain variability and document telemetry deltas for with/without rain comparisons.

**Exit Criteria**

* Regional rain visibly modulates fire and speeds lava cooling.
* Tuning pass yields cycles of growth → fire/eruption → recovery that look natural.

---

## Cross-Cutting: QA, Debuggability, Determinism

* **Golden seeds:** keep a small set of seeds + parameter packs;
* **Determinism gate:** any change must reproduce golden outputs unless parameters changed.

---

## Suggested Milestones (quick view)

1. **M1:** Layers + masks + params load (Phase 1).
2. **M2:** Vegetation spread/succession stable (Phase 2).
3. **M3:** Fire lifecycle working, manual ignition (Phase 3).
4. **M4:** Volcano proto→eruption + lava dynamics (Phase 4).
5. **M5:** Rain regions wired; tuning & polish (Phase 5).
