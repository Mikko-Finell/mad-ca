package life

import "testing"

func TestBlinkerOscillation(t *testing.T) {
	life := New(5, 5)
	cells := life.Cells()
	for i := range cells {
		cells[i] = 0
	}

	w := life.Size().W
	set := func(x, y int) { life.Cells()[y*w+x] = 1 }
	set(2, 1)
	set(2, 2)
	set(2, 3)

	life.Step()
	cells = life.Cells()

	expects := map[[2]int]bool{
		{1, 2}: true,
		{2, 2}: true,
		{3, 2}: true,
	}

	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			idx := y*w + x
			alive := cells[idx] == 1
			_, shouldBeAlive := expects[[2]int{x, y}]
			if shouldBeAlive != alive {
				t.Fatalf("cell (%d,%d) alive=%v, expected %v", x, y, alive, shouldBeAlive)
			}
		}
	}

	life.Step()
	cells = life.Cells()

	expects = map[[2]int]bool{
		{2, 1}: true,
		{2, 2}: true,
		{2, 3}: true,
	}

	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			idx := y*w + x
			alive := cells[idx] == 1
			_, shouldBeAlive := expects[[2]int{x, y}]
			if shouldBeAlive != alive {
				t.Fatalf("after second step cell (%d,%d) alive=%v, expected %v", x, y, alive, shouldBeAlive)
			}
		}
	}
}
