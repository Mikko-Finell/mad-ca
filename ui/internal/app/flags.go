package app

import "flag"

// Config represents the command-line parameters for the application.
type Config struct {
	Sim   string
	Scale int
	TPS   int
	Seed  int64
}

// NewConfig returns a Config populated with sensible defaults.
func NewConfig() *Config {
	return &Config{Sim: "life", Scale: 3, TPS: 60, Seed: 42}
}

// Bind attaches the configuration to the provided FlagSet.
func (c *Config) Bind(fs *flag.FlagSet) {
	fs.StringVar(&c.Sim, "sim", c.Sim, "simulation to run")
	fs.IntVar(&c.Scale, "scale", c.Scale, "pixel scale multiplier")
	fs.IntVar(&c.TPS, "tps", c.TPS, "ticks per second")
	fs.Int64Var(&c.Seed, "seed", c.Seed, "seed for simulation reset")
}
