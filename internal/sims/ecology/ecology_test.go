package ecology

import "testing"

func TestWorldRemainsEmpty(t *testing.T) {
	world := New(4, 4)
	for i := range world.cells {
		world.cells[i] = 1
	}

	world.Reset(42)
	world.Step()

	for i, cell := range world.cells {
		if cell != 0 {
			t.Fatalf("cell %d = %d, want 0", i, cell)
		}
	}
}
