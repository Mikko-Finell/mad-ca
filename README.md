# mad-ca

mad-ca is a modular cellular automata playground built with [Ebitengine](https://ebiten.org/). It bundles a reusable renderer, a simple
application shell, and a growing catalog of simulation rules. Each simulation is implemented as a pure Go package that satisfies a small
`core.Sim` interface, which keeps logic easy to test and reuse outside of the rendering layer.

## Getting started

```bash
go run -tags ebiten ./cmd/ca -sim=life -scale=3 -tps=60
```

### Auto-sync dev loop

`make run` and the sim-specific targets (for example `make ecology` or
`make briansbrain`) delegate to `scripts/devsync.sh`. The helper keeps the
working tree aligned with the branch you have checked out, rebuilds the binary,
and restarts the simulation automatically whenever the upstream branch changes.
Override the defaults by exporting environment variables before invoking the
target:

```bash
# follow a feature branch and poll every 5 seconds
BRANCH=my-feature POLL_SECONDS=5 make ecology
```

Without overrides the script follows the branch currently checked out in your
working tree, so `make life` continues polling the same branch you are editing.

Pass additional simulation flags directly to the Make target; they will be
forwarded to the built binary by the helper script.

> **Note**
>
> The graphical build depends on native GLFW/X11 headers. When those headers are
> unavailable (as in many CI or container environments) the repository falls
> back to a headless stub so `go test ./...` continues to work. To run the GUI
> you must pass the `ebiten` build tag as shown above.

## Project layout

The repository follows a layered structure:

- `internal/core` exposes the foundational types (`Sim`, `Size`, timers, RNG helpers).
- `internal/app` owns the Ebitengine `Game` adapter and command-line flag parsing.
- `internal/render` provides efficient pixel upload helpers for grid-based simulations.
- `internal/sims/*` contains self-contained implementations of individual simulations (Game of Life, Brian's Brain, Elementary rules, Ecology placeholder).
- `internal/ui` is reserved for optional overlays (FPS counters, controls, etc.).
- `assets` stores fonts, images, and shaders that can be embedded into the binary.
- `pkg` is for code that could be reused outside of the application (currently empty).

Refer to `Makefile` for common tasks such as running, building, linting, or targeting WebAssembly.
