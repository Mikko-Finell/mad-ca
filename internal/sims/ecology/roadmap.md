# Cellular-Ecology CA — Implementation Roadmap (5 Phases)

## Phase 1 — Model & Config Wiring

**Status:** Complete — deterministic world scaffolding in place and rain/volcano mask overlays renderable via debug toggles (`1` rain, `2` volcano). Ready to begin vegetation dynamics work.

**Goal:** Represent the world state and parameters, no behavior yet.

**Scope**

* Define two layers: **Ground** (`Dirt|Rock|Mountain|Lava`) and **Vegetation** (`None|Grass|Shrub|Tree`).
* Add per-cell auxiliaries: lava thickness/temperature/direction/tip flags plus `burn_ttl:int` (0 = inactive).
* Introduce **region masks** (float [0..1]): `RainMask`, `VolcanoMask` (double-buffered).
* Load a static `tectonic_map` ([0..1]) and a deterministic PRNG (seeded).
* Centralize tunables (thresholds/probabilities/lifetimes) in a params object.

**Tests & Tools**

* Determinism: same seed → identical layers after N ticks (no rules yet).

**Exit Criteria**

* World can be initialized (Dirt + Rock sprinkles + Grass seeds + tectonic_map).
* Params adjustable at runtime; masks exist and are rendered as overlays.

**Notes:**

* HUD parameter snapshot plumbing now surfaces the full config to the app so future controls can adjust values without poking the simulation internals, and the HUD now offers +/- controls for the primary ecology tunables (lava, fire, rain, vegetation) so they can be adjusted live.

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
* Default grass spread chance now starts at 1%, leaving ample room to tune early meadow growth while still allowing gradual colonization.

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

**Status:** Complete — proto-volcano region spawner, mask rasterization, uplift conversion, and the lava system has been upgraded to branching rivers with vent-fed flows, channel memory, and rain-aware cooling.

**Goal:** Geological engine: uplift, eruptions, lava spread/cool.

**Scope**

* **Region events framework:** list of active regions (center, radius, falloff, ttl, noise seed); per-tick rasterization to `VolcanoMask`. *(Volcano protoregion list implemented with linear falloff masks; expirations tracked for future eruption handling.)*
* **Proto-volcano uplift:** Rock→Mountain with chance scaled by `VolcanoMask * P_uplift_base`. *(Implemented — uplift now occurs after mask build.)*
* **Eruption on expiry:** chance `P_erupt_base * mean(VolcanoMask in region)`; write core lava, rim mountains, occasional lava specks.
* **Lava:** vent-driven rivers with per-cell `(h,T,dir,tip,channel)` state, tip-based advancement, pooling/overflow handling, and cooling/crusting that solidifies to rock when fronts chill.
* keep order: build masks → uplift/erupt → lava spread/cool → fire → vegetation. *(Tick now builds volcano masks before uplifting and the existing lava/fire/vegetation phases.)*

**Tests & Telemetry**

* Volcano lifecycle: proto appears → uplift rises → eruption sometimes occurs → lava cools to rock.
* Spread bounded (one-ring per tick) with double buffering—no mid-tick cascades.
* Long-run equilibrium check: repeated cycles don’t crash performance or blow up memory.

**Exit Criteria**

* Volcano protos form roundish uplift, eruptions produce credible cones/lava fields.
* Lava cools reliably; no stuck eternal lava without config asking for it.

**Notes:**

* Proto-volcano lifecycle now consumes proto expirations to trigger eruptions that seed lava cores, uplift rims, and clear vegetation. Manual volcano spawns now erupt instantly so debug placements can build a lava field without waiting on proto timers. The lava subsystem now carries thickness/temperature/heading data, advances tips along slopes with channel reinforcement, handles pooling and crusting, and vents maintain core flux. Rain both penalizes forward scoring and boosts cooling, and new tests cover vent seeding, cooling/crusting, channel reinforcement, and lava-driven ignition.
* HUD exposes an adjustable volcano eruption base chance so tuning sessions can readily force eruptions when needed.
* Lava persistence now leans on reservoir-fed vents and flux-aware cooling so crater pools linger while flowing rivers stay molten. The HUD snapshot exposes the new cooling and reservoir parameters for future tuning passes.
* **Follow-up:** Focus on river shaping—channel-biased flow scoring, smarter overflow spokes, and telemetry for reservoir mass/flux to validate the new persistence model.
* Latest crater rebuild pass carves 1–3 random spillway openings whose floor matches the reservoir head so vents keep feeding once the starter pool drains along those exits.
* HUD cleanup removed the redundant “Ecology Controls” banner and now surfaces direct adjustments for lava cooling, flux reference, reservoir head, and proto strength/uplift so lava longevity and volcano shaping can be tuned live.
* Added a debug overlay toggle (`4`) that shades the lava elevation raster so eruption tuning sessions can see crater rims and spillways at a glance.
* Eruptions now clear only their local footprint, so spawning a new volcano no longer wipes out existing lava fields or vents elsewhere on the map.
* Shift-clicking the map now drops a proto-volcano at the cursor, giving tuning sessions a deterministic way to raise cones without waiting for random spawns.
* Lava elevation overlays now persist across eruptions, clamp overlap by taking the per-tile maximum, and limit new craters to influencing terrain within one diameter beyond the rim so older cones remain visible when new vents appear.
* Refined the lava elevation raster bounds so the crater influence window matches the intended radius and added extra index guards while writing elevations.

---

## Phase 5 — Regional Rain Integration & Tuning

**Status:** Complete — tuned regional rain spawning/strength to recover pre-rain lava variability, added environment telemetry for rain vs. dry runs, and regression tests now cover the contrast.

**Goal:** Replace per-tile wetness with **Rain regions** and wire all multipliers.

**Scope**

* Rain regions rasterized to `RainMask` each tick (gaussian falloff, 8–24 radius, 9–18 ttl).
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
* Tuned rain spawn chance/strength ranges to keep dry spells between storms, introduced environment telemetry helpers to summarize ground/fire/rain state, and regression runs now assert that rainy cycles cool lava faster while keeping large eruption oscillations.
* Display pipeline now encodes ground, vegetation, and burning states into a palette-backed buffer so the sim’s cycles are visible in the app; palette coverage verified by new tests.
* Palette entries are cached as `[]color.RGBA` to avoid per-pixel interface conversions while blitting the display buffer.
* Rain masks now render drifting noise-shaped cloud blobs with inertia-smooth drift, coherent neighborhood flow, stabilized silhouettes, and strengthened morphology cleanup to eliminate spray artifacts; documentation updated to match.
* Latest tuning lowered the noise gate to τ≈0.35–0.45 with a smoothstep blend, enforces solid cores, and widens the morphology closing radius to 2px to plug noise pinholes.
* HUD now surfaces wind noise and speed controls so storm drift can be dialed in live during tuning sessions.
* Added a HUD slider for the wind temporal scale so the curl-noise phase spin can be slowed during tuning while keeping the default value near the top of the range to match prior visuals.
* HUD renders a wind vector overlay to visualize current drift averages for active storm regions.
* Rain drift and the HUD overlay now sample a single world-seed wind field (curl of an fBm potential), so every storm follows the same streamlines the overlay depicts.
* HUD parameter buttons now auto-scale their step sizes, present chance values as 0–100%, and no longer clamp tuning ranges with arbitrary ceilings.

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
