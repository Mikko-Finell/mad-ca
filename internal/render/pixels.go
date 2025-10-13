package render

import "image/color"

// fillBinaryRGBA converts binary cell data (0/1) into RGBA pixels in buf.
func fillBinaryRGBA(buf []byte, cells []uint8, on, off color.Color) {
	rOn, gOn, bOn, aOn := on.RGBA()
	rOff, gOff, bOff, aOff := off.RGBA()
	for i, c := range cells {
		base := i * 4
		if c != 0 {
			buf[base+0] = uint8(rOn >> 8)
			buf[base+1] = uint8(gOn >> 8)
			buf[base+2] = uint8(bOn >> 8)
			buf[base+3] = uint8(aOn >> 8)
			continue
		}
		buf[base+0] = uint8(rOff >> 8)
		buf[base+1] = uint8(gOff >> 8)
		buf[base+2] = uint8(bOff >> 8)
		buf[base+3] = uint8(aOff >> 8)
	}
}

// fillPaletteRGBA converts cell values into RGBA pixels using a palette. When
// the palette is empty the buffer is cleared to transparent black.
func fillPaletteRGBA(buf []byte, cells []uint8, palette []color.RGBA) {
	if len(palette) == 0 {
		for i := range cells {
			base := i * 4
			buf[base+0] = 0
			buf[base+1] = 0
			buf[base+2] = 0
			buf[base+3] = 0
		}
		return
	}

	last := len(palette) - 1
	for i, c := range cells {
		idx := int(c)
		if idx > last {
			idx = last
		}
		base := i * 4
		col := palette[idx]
		buf[base+0] = col.R
		buf[base+1] = col.G
		buf[base+2] = col.B
		buf[base+3] = col.A
	}
}
