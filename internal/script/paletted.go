package script

import (
	"fmt"
	"image"
	"image/color"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

var screenRect = image.Rect(0, 0, 640, 480)

// Paletted is a Canvas of palette indices at the original 640x480, drawing
// the way the 1992 game did. It needs no graphics card, so tests use it,
// and it is the reference the scaled renderer is checked against.
type Paletted struct {
	Origin  image.Point // added to all coordinates
	Palette color.Palette

	archive    *resource.Archive
	bitmaps    map[string][]Sprite
	background []uint8
	bgHeight   int // rows covered by the background; the rest is black
	saved      *plane
	layers     [MaxThreads]*plane
}

// NewPaletted returns a Paletted canvas loading pictures from a.
func NewPaletted(a *resource.Archive, pal color.Palette) *Paletted {
	p := &Paletted{
		Palette:    pal,
		archive:    a,
		bitmaps:    map[string][]Sprite{},
		background: make([]uint8, 640*480),
	}
	p.saved = p.newPlane()
	for i := range p.layers {
		p.layers[i] = p.newPlane()
		p.layers[i].hidden = true
	}
	return p
}

func (p *Paletted) newPlane() *plane {
	return &plane{pix: make([]uint8, 640*480), clip: screenRect, origin: &p.Origin}
}

// LoadBitmap implements Canvas. Its sprites are *resource.Image values.
func (p *Paletted) LoadBitmap(name string) ([]Sprite, error) {
	if s, ok := p.bitmaps[name]; ok {
		return s, nil
	}
	e := p.archive.Find(name)
	if e == nil {
		return nil, fmt.Errorf("%s not found", name)
	}
	b, err := resource.DecodeBMP(e)
	if err != nil {
		return nil, err
	}
	s := make([]Sprite, len(b.Frames))
	for i := range b.Frames {
		s[i] = &b.Frames[i]
	}
	p.bitmaps[name] = s
	return s, nil
}

// LoadScreen implements Canvas.
func (p *Paletted) LoadScreen(name string) error {
	e := p.archive.Find(name)
	if e == nil {
		return fmt.Errorf("%s not found", name)
	}
	img, err := resource.DecodeSCR(e)
	if err != nil {
		return err
	}
	p.SetBackground(img)
	p.ClearSaved()
	return nil
}

// SetBackground replaces the background with img, at most 640x480.
func (p *Paletted) SetBackground(img *resource.Image) {
	clear(p.background)
	w, h := min(img.Width, 640), min(img.Height, 480)
	for y := range h {
		copy(p.background[y*640:y*640+w], img.Pix[y*img.Width:])
	}
	p.bgHeight = h
}

// Layer implements Canvas.
func (p *Paletted) Layer(i int) Layer { return p.layers[i] }

// SaveZone implements Canvas.
func (p *Paletted) SaveZone(l Layer, x, y, w, h int) {
	src := l.(*plane)
	r := image.Rect(x, y, x+w, y+h).Add(p.Origin).Intersect(screenRect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c := src.pix[y*640+x]; c != 0 {
				p.saved.pix[y*640+x] = c
			}
		}
	}
}

// ClearSaved implements Canvas.
func (p *Paletted) ClearSaved() { p.saved.Clear() }

// Compose returns the picture as shown: the background, the saved zones,
// then the visible layers in slot order. Below a background shorter than
// 480 rows the screen is black.
func (p *Paletted) Compose() *image.NRGBA {
	planes := []*plane{p.saved}
	for _, l := range p.layers {
		if !l.hidden {
			planes = append(planes, l)
		}
	}
	img := image.NewNRGBA(screenRect)
	for i, c := range p.background {
		shown := i < p.bgHeight*640
		for _, l := range planes {
			if l.pix[i] != 0 {
				c, shown = l.pix[i], true
			}
		}
		img.Pix[4*i+3] = 0xFF
		if shown {
			rgba := color.NRGBAModel.Convert(p.Palette[c]).(color.NRGBA)
			img.Pix[4*i+0], img.Pix[4*i+1], img.Pix[4*i+2] = rgba.R, rgba.G, rgba.B
		}
	}
	return img
}

// plane is a Paletted layer. Index 0 is transparent.
type plane struct {
	pix    []uint8
	clip   image.Rectangle
	hidden bool
	origin *image.Point
}

func (l *plane) Clear()            { clear(l.pix) }
func (l *plane) SetVisible(v bool) { l.hidden = !v }
func (l *plane) ResetClip()        { l.clip = screenRect }

func (l *plane) SetClip(x1, y1, x2, y2 int) {
	l.clip = image.Rect(x1, y1, x2, y2).Add(*l.origin).Intersect(screenRect)
}

func (l *plane) FillRect(x, y, w, h int, c uint8) {
	r := image.Rect(x, y, x+w, y+h).Add(*l.origin).Intersect(l.clip)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := l.pix[y*640:]
		for x := r.Min.X; x < r.Max.X; x++ {
			row[x] = c
		}
	}
}

func (l *plane) DrawSprite(s Sprite, x, y int, flip bool) {
	img := s.(*resource.Image)
	x += l.origin.X
	y += l.origin.Y
	r := image.Rect(x, y, x+img.Width, y+img.Height).Intersect(l.clip)
	for py := r.Min.Y; py < r.Max.Y; py++ {
		src := img.Pix[(py-y)*img.Width:]
		for px := r.Min.X; px < r.Max.X; px++ {
			sx := px - x
			if flip {
				sx = img.Width - 1 - sx
			}
			if c := src[sx]; c != 0 {
				l.pix[py*640+px] = c
			}
		}
	}
}
