package resource

import (
	"fmt"
	"image/color"
)

// Image is a 16-color picture as palette indices, row by row.
type Image struct {
	Width, Height int
	Pix           []uint8
}

// unpack4 reads 4-bit pixels, high nibble first, starting at nibble index
// start of data. It returns the image and the nibble index after it.
func unpack4(data []byte, start, w, h int) (Image, int, error) {
	n := w * h
	if start+n > len(data)*2 {
		return Image{}, 0, fmt.Errorf("%dx%d image needs %d pixels, %d left", w, h, n, len(data)*2-start)
	}
	img := Image{Width: w, Height: h, Pix: make([]uint8, n)}
	for i := range n {
		b := data[(start+i)/2]
		if (start+i)%2 == 0 {
			img.Pix[i] = b >> 4
		} else {
			img.Pix[i] = b & 0x0F
		}
	}
	return img, start + n, nil
}

// Sprites is a BMP resource: a sheet of sprite frames.
type Sprites struct {
	Frames []Image
	// VGA holds the upper 4 bits of each pixel for 256-color mode, packed
	// like the frames, or nil. Only JOHNWALK.BMP and MJTELE.BMP have it, with
	// six non-zero nibbles each. The 16-color game ignores it, and so do we.
	VGA []byte
}

// DecodeBMP decodes a BMP resource.
//
// Layout: "BMP:" u16 width u16 height (unused), "INF:" u32 size u16 count,
// count u16 widths, count u16 heights, "BIN:" packed 4-bit pixels of all
// frames in order, optionally "VGA:" packed upper pixel bits.
func DecodeBMP(e *Entry) (*Sprites, error) {
	r := newReader(e.Name, e.Data)
	r.tag("BMP:")
	r.u16()
	r.u16()
	r.tag("INF:")
	r.u32()
	count := int(r.u16())
	widths := make([]int, count)
	heights := make([]int, count)
	for i := range widths {
		widths[i] = int(r.u16())
	}
	for i := range heights {
		heights[i] = int(r.u16())
	}
	r.tag("BIN:")
	data := r.packed()
	var vga []byte
	if r.err == nil && r.remaining() > 0 {
		r.tag("VGA:")
		vga = r.packed()
		if r.err == nil && len(vga) != len(data) {
			return nil, fmt.Errorf("%s: VGA block is %d bytes, BIN block %d", e.Name, len(vga), len(data))
		}
	}
	if err := r.done(); err != nil {
		return nil, err
	}

	s := &Sprites{Frames: make([]Image, count), VGA: vga}
	pos := 0
	for i := range count {
		img, next, err := unpack4(data, pos, widths[i], heights[i])
		if err != nil {
			return nil, fmt.Errorf("%s frame %d: %w", e.Name, i, err)
		}
		s.Frames[i] = img
		pos = next
	}
	// At most one padding nibble may follow the last frame.
	if left := len(data)*2 - pos; left > 1 {
		return nil, fmt.Errorf("%s: %d unused pixels after the last frame", e.Name, left)
	}
	return s, nil
}

// DecodeSCR decodes an SCR resource: a full-screen background.
//
// Layout: "SCR:" u16 u16 (unused), "DIM:" u32 size u16 width u16 height,
// "BIN:" packed 4-bit pixels.
func DecodeSCR(e *Entry) (*Image, error) {
	r := newReader(e.Name, e.Data)
	r.tag("SCR:")
	r.u16()
	r.u16()
	r.tag("DIM:")
	r.u32()
	w := int(r.u16())
	h := int(r.u16())
	r.tag("BIN:")
	data := r.packed()
	if err := r.done(); err != nil {
		return nil, err
	}
	img, end, err := unpack4(data, 0, w, h)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", e.Name, err)
	}
	if end != len(data)*2 {
		return nil, fmt.Errorf("%s: %d unused pixels", e.Name, len(data)*2-end)
	}
	return &img, nil
}

// Palette is a VGA palette with 6-bit components (0-63).
type Palette [256][3]uint8

// Color returns entry i as 8-bit RGB, scaling 0-63 to 0-255.
func (p *Palette) Color(i int) color.NRGBA {
	c := p[i]
	scale := func(v uint8) uint8 { return v<<2 | v>>4 }
	return color.NRGBA{R: scale(c[0]), G: scale(c[1]), B: scale(c[2]), A: 0xFF}
}

// DecodePAL decodes a PAL resource.
//
// Layout: "PAL:" u16 u8 u8 (unused), "VGA:" 4 bytes (unused), 256 RGB
// triplets.
func DecodePAL(e *Entry) (*Palette, error) {
	r := newReader(e.Name, e.Data)
	r.tag("PAL:")
	r.u16()
	r.u8()
	r.u8()
	r.tag("VGA:")
	r.bytes(4)
	var p Palette
	for i := range p {
		copy(p[i][:], r.bytes(3))
	}
	if err := r.done(); err != nil {
		return nil, err
	}
	for i, c := range p {
		if c[0] > 63 || c[1] > 63 || c[2] > 63 {
			return nil, fmt.Errorf("%s: color %d (%v) is not 6-bit", e.Name, i, c)
		}
	}
	return &p, nil
}
