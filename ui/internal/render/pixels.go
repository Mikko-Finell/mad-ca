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
