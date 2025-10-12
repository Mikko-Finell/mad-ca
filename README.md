# mad-ca

mad-ca is a modular cellular automata playground built with [Ebitengine](https://ebiten.org/). It bundles a reusable renderer, a simple
application shell, and a growing catalog of simulation rules. Each simulation is implemented as a pure Go package that satisfies a small
`core.Sim` interface, which keeps logic easy to test and reuse outside of the rendering layer.

The repository is split into two Go modules:

* the root module (`mad-ca`) hosts reusable simulation logic and utility packages that are fully testable without any graphics dependencies;
* the `ui` module (`mad-ca/ui`) contains the Ebiten-powered desktop application that depends on the root module.

## Getting started

```bash
cd ui
go run ./cmd/ca -sim=life -scale=3 -tps=60
```

## Project layout

The repository follows a layered structure:

- `pkg/core` exposes the foundational types (`Sim`, `Size`, timers, RNG helpers`).
- `pkg/sims/*` contains self-contained implementations of individual simulations (Game of Life, Brian's Brain, Elementary rules).
- `pkg/caio` keeps data shuttles for simulation IO.
- `ui/internal/app` owns the Ebitengine `Game` adapter and command-line flag parsing.
- `ui/internal/render` provides efficient pixel upload helpers for grid-based simulations.
- `ui/internal/ui` is reserved for optional overlays (FPS counters, controls, etc.).
- `assets` stores fonts, images, and shaders that can be embedded into the binary.
- `pkg` is for code that could be reused outside of the application (currently empty).

Refer to `Makefile` for common tasks such as running, building, linting, or targeting WebAssembly.
