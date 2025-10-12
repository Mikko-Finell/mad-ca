# mad-ca

mad-ca is a modular cellular automata playground built with [Ebitengine](https://ebiten.org/). It bundles a reusable renderer, a simple
application shell, and a growing catalog of simulation rules. Each simulation is implemented as a pure Go package that satisfies a small
`core.Sim` interface, which keeps logic easy to test and reuse outside of the rendering layer.

## Getting started

```bash
go run ./cmd/ca -sim=life -scale=3 -tps=60
```

## Project layout

The repository follows a layered structure:

- `internal/core` exposes the foundational types (`Sim`, `Size`, timers, RNG helpers).
- `internal/app` owns the Ebitengine `Game` adapter and command-line flag parsing.
- `internal/render` provides efficient pixel upload helpers for grid-based simulations.
- `internal/sims/*` contains self-contained implementations of individual simulations (Game of Life, Brian's Brain, Elementary rules).
- `internal/ui` is reserved for optional overlays (FPS counters, controls, etc.).
- `assets` stores fonts, images, and shaders that can be embedded into the binary.
- `pkg` is for code that could be reused outside of the application (currently empty).

Refer to `Makefile` for common tasks such as running, building, linting, or targeting WebAssembly.
