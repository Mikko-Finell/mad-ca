package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// GridPainter updates a single RGBA image based on binary cell data.
type GridPainter struct {
	w, h int
	img  *ebiten.Image
	buf  []byte
}

// NewGridPainter allocates a painter for a grid of size w*h.
func NewGridPainter(w, h int) *GridPainter {
	gp := &GridPainter{w: w, h: h, buf: make([]byte, 4*w*h)}
	gp.img = ebiten.NewImage(w, h)
	return gp
}

// Blit uploads the provided cells into the painter image and draws it.
func (gp *GridPainter) Blit(dst *ebiten.Image, cells []uint8, on, off color.Color, scale int) {
	if len(cells) != gp.w*gp.h {
		return
	}
	fillBinaryRGBA(gp.buf, cells, on, off)
	gp.img.ReplacePixels(gp.buf)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	dst.DrawImage(gp.img, op)
}

// Size returns the dimensions of the underlying image.
func (gp *GridPainter) Size() (int, int) { return gp.w, gp.h }
