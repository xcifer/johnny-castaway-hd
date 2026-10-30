package main

import (
	"fmt"
	"image"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// previewCrop is the part of the title picture the preview shows: the
// scroll with the title, without the black around it, in 4:3.
var previewCrop = image.Rect(100, 70, 510, 378)

// previewImage is the original's title picture, INTRO.SCR, which the
// preview in Screen Saver Settings shows.
func previewImage(dir string) (*image.RGBA, error) {
	a, err := resource.Load(dir)
	if err != nil {
		return nil, err
	}
	scr, pal := a.Find("INTRO.SCR"), a.Find("JOHNCAST.PAL")
	if scr == nil || pal == nil {
		return nil, fmt.Errorf("INTRO.SCR or JOHNCAST.PAL not found")
	}
	img, err := resource.DecodeSCR(scr)
	if err != nil {
		return nil, err
	}
	p, err := resource.DecodePAL(pal)
	if err != nil {
		return nil, err
	}
	out := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	for i, c := range img.Pix {
		col := p.Color(int(c))
		copy(out.Pix[4*i:], []uint8{col.R, col.G, col.B, 0xFF})
	}
	return out, nil
}
