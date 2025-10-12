package core

// Size describes the dimensions of a simulation grid.
type Size struct {
	W int
	H int
}

// Sim defines the minimal contract a cellular automaton must implement.
type Sim interface {
	Name() string
	Size() Size
	Reset(seed int64)
	Step()
	Cells() []uint8
}

// Factory constructs a Sim using an optional configuration map.
type Factory func(cfg map[string]string) Sim

var sims = map[string]Factory{}

// Register adds a simulation factory under the provided name.
func Register(name string, f Factory) {
	if name == "" || f == nil {
		return
	}
	sims[name] = f
}

// Sims exposes the registry of available simulation factories.
func Sims() map[string]Factory {
	return sims
}
