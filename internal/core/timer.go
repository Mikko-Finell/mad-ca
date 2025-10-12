package core

import "time"

// FixedStep helps run simulation updates at a steady ticks-per-second rate.
type FixedStep struct {
	step        time.Duration
	accumulator time.Duration
	last        time.Time
}

// NewFixedStep constructs a FixedStep controller targeting the given TPS.
func NewFixedStep(tps int) *FixedStep {
	if tps <= 0 {
		tps = 60
	}
	fs := &FixedStep{}
	fs.SetTPS(tps)
	fs.accumulator = fs.step
	return fs
}

// SetTPS changes the tick rate. It is safe to call from the main loop.
func (f *FixedStep) SetTPS(tps int) {
	if tps <= 0 {
		tps = 60
	}
	f.step = time.Second / time.Duration(tps)
}

// ShouldStep reports whether the simulation should advance by one tick.
func (f *FixedStep) ShouldStep() bool {
	now := time.Now()
	if f.last.IsZero() {
		f.last = now
	}
	delta := now.Sub(f.last)
	f.last = now
	f.accumulator += delta
	if f.accumulator >= f.step {
		f.accumulator -= f.step
		return true
	}
	return false
}
