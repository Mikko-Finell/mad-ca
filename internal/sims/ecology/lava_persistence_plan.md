# Lava Persistence & River Formation Plan

## Status

* ✅ Flux-weighted cooling is live: stationary pools no longer receive an arbitrary penalty. Cooling now scales with rain, edge exposure, thickness via `σ(h) = 1 − e^{−h}`, and a `(1 − q_out/LavaFluxRef)` flux term so active rivers stay hot while stagnant pools crust faster.【F:internal/sims/ecology/ecology.go†L2805-L2864】
* ✅ Vents are reservoir-fed: each eruption seeds vents with sampled `massRemaining`, head, and gain. Injection now drains the reservoir instead of counting down a TTL, and depletion cleanly removes vents.【F:internal/sims/ecology/ecology.go†L2398-L2487】【F:internal/sims/ecology/config.go†L64-L113】
* ✅ Thermal hysteresis caps reheated crust at `min(Tc + Teps, lavaReheatCap)` so newly cooled columns linger before re-melting, preventing flicker while keeping vents responsive.【F:internal/sims/ecology/ecology.go†L2838-L2859】

## Current Mechanics Snapshot

1. **Injection:** Per-tick vent flux is `ceil(Kp × max(0, head − surface))`, limited by remaining reservoir mass and the per-tile height cap (7). The vent tile reheats to 1.0 and refreshes the outlet tip on each successful pump.【F:internal/sims/ecology/ecology.go†L2398-L2487】
2. **Flux tracking:** Every source cell that advances or splits records the integer mass removed in `q_out`. Cooling consumes this log then resets it, so only tiles that actually discharged receive the low-cooling benefit.【F:internal/sims/ecology/ecology.go†L2552-L2675】【F:internal/sims/ecology/ecology.go†L2805-L2864】
3. **Crusting:** When the cooled temperature drops below `Tc`, tall columns shed one unit (temperature capped at `Tc + Teps`); a height-1 column solidifies to rock and clears lava state. Flux queues reset at the end of cooling, ready for the next tick.【F:internal/sims/ecology/ecology.go†L2830-L2864】

## Next Steps

### 1. River encouragement

1. Use the existing `lavaChannel` field as a semi-permanent lava viscosity reduction. Lower the denominator in the movement chance (`1 + lavaSpeedAlpha * height`) when channel weight is high, making carved channels accelerate flow.
2. When selecting candidates in `spawnLavaChild`, add a concavity bias by sampling the elevation difference within a 3×3 stencil. Favor cells where the downstream 2-step average is strictly lower than adjacent diagonals—mimicking natural incision.
3. Track recent split attempts; suppress splits that would create two nearly equal scores unless the parent column has `lavaForce` or height ≥ 4. This funnels lava into fewer, deeper rivers.
4. Allow pooled lava around the crater to overflow preferentially along low-elevation spokes by looking two tiles out before filling a neighbor. This encourages discrete outlets instead of a uniform rim breach.

### 2. Parameter & tooling updates

1. Expose new config knobs:
   * Channel shaping toggles (e.g. `LavaChannelFlowBoost`, concavity weight) once river bias work lands.
   * Flux telemetry scaling (e.g. HUD display of 80th percentile flux) if adaptive cooling proves useful.
2. Update HUD parameter snapshot to surface the new knobs and provide quick multipliers (e.g. ×0.5, ×2) to explore persistence ranges.
3. Extend lava telemetry to chart active reservoir mass and mean outlet flux to confirm longevity tuning.

### 3. Testing

* Unit tests:
  * Validate that high-flux rivers retain higher mean temperature than low-flux pools after a fixed number of ticks.
  * Confirm that vents deplete `massRemaining` and deactivate once exhausted (in place via `TestLavaReservoirDepletesVent`).
  * Ensure rivers still advance downhill on varying slopes despite the new concavity bias.
* Regression tests:
  * Golden-seed comparison to assert lava footprint area remains within bounds after 200 ticks (no infinite floods).
  * Add a scenario where rain is disabled to observe longer-lived crater pools compared to baseline snapshots.
