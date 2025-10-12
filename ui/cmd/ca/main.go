package main

import (
	"errors"
	"flag"
	"log"

	"mad-ca/pkg/core"
	_ "mad-ca/pkg/sims/briansbrain"
	_ "mad-ca/pkg/sims/elementary"
	_ "mad-ca/pkg/sims/life"
	"mad-ca/ui/internal/app"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	cfg := app.NewConfig()
	cfg.Bind(flag.CommandLine)
	flag.Parse()

	factory, ok := core.Sims()[cfg.Sim]
	if !ok {
		log.Fatalf("unknown sim %q", cfg.Sim)
	}

	sim := factory(nil)
	sim.Reset(cfg.Seed)

	game := app.New(sim, cfg.Scale, cfg.Seed)
	size := sim.Size()

	ebiten.SetWindowTitle("mad-ca — " + sim.Name())
	ebiten.SetTPS(cfg.TPS)
	ebiten.SetWindowSize(size.W*cfg.Scale, size.H*cfg.Scale)

	if err := ebiten.RunGame(game); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}
