package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/xcifer/johnny-castaway-hd/internal/render"
	"github.com/xcifer/johnny-castaway-hd/internal/resource"
	"github.com/xcifer/johnny-castaway-hd/internal/script"
)

// screenCanvas lets scripts draw through the renderer, moved by origin:
// where the island is, relative to where the scripts expect it.
type screenCanvas struct {
	r       *render.Renderer
	d       *gameData
	origin  image.Point
	layers  [script.MaxThreads]*canvasLayer
	bitmaps map[string][]script.Sprite
	screens map[string]*resource.Image
	// screenLoaded is called when a script replaces the background.
	screenLoaded func()
}

func newScreenCanvas(r *render.Renderer, d *gameData) *screenCanvas {
	c := &screenCanvas{r: r, d: d, bitmaps: map[string][]script.Sprite{}, screens: map[string]*resource.Image{}}
	for i := range c.layers {
		l := r.NewLayer()
		l.SetVisible(false)
		c.layers[i] = &canvasLayer{c: c, l: l}
	}
	return c
}

func (c *screenCanvas) LoadBitmap(name string) ([]script.Sprite, error) {
	if s, ok := c.bitmaps[name]; ok {
		return s, nil
	}
	e := c.d.archive.Find(name)
	if e == nil {
		return nil, fmt.Errorf("%s not found", name)
	}
	b, err := resource.DecodeBMP(e)
	if err != nil {
		return nil, err
	}
	s := make([]script.Sprite, len(b.Frames))
	for i := range b.Frames {
		s[i] = render.NewSprite(&b.Frames[i], c.d.palette, true)
	}
	c.bitmaps[name] = s
	return s, nil
}

func (c *screenCanvas) LoadScreen(name string) error {
	img, ok := c.screens[name]
	if !ok {
		e := c.d.archive.Find(name)
		if e == nil {
			return fmt.Errorf("%s not found", name)
		}
		var err error
		if img, err = resource.DecodeSCR(e); err != nil {
			return err
		}
		c.screens[name] = img
	}
	c.r.SetBackground(c.d.screen(img))
	c.r.Saved.Clear()
	if c.screenLoaded != nil {
		c.screenLoaded()
	}
	return nil
}

func (c *screenCanvas) Layer(i int) script.Layer { return c.layers[i] }

func (c *screenCanvas) SaveZone(l script.Layer, x, y, w, h int) {
	c.r.Origin = c.origin
	c.r.CopyZoneToSaved(l.(*canvasLayer).l, x, y, w, h)
}

func (c *screenCanvas) ClearSaved() { c.r.Saved.Clear() }

// clear removes everything the scripts have drawn.
func (c *screenCanvas) clear() {
	for _, l := range c.layers {
		l.l.Clear()
	}
	c.r.Saved.Clear()
}

// canvasLayer is a script layer drawn by the renderer.
type canvasLayer struct {
	c *screenCanvas
	l *render.Layer
}

func (l *canvasLayer) Clear()            { l.l.Clear() }
func (l *canvasLayer) SetVisible(v bool) { l.l.SetVisible(v) }
func (l *canvasLayer) SetClip(x1, y1, x2, y2 int) {
	l.c.r.Origin = l.c.origin
	l.c.r.SetClip(l.l, x1, y1, x2, y2)
}

func (l *canvasLayer) ResetClip() { l.c.r.ResetClip(l.l) }

func (l *canvasLayer) DrawSprite(s script.Sprite, x, y int, flip bool) {
	l.c.r.Origin = l.c.origin
	l.c.r.DrawSprite(l.l, s.(*render.Sprite), x, y, flip)
}

// FillRect fills with palette color c; color 0 erases.
func (l *canvasLayer) FillRect(x, y, w, h int, c uint8) {
	var col color.Color = color.Transparent
	if c != 0 {
		col = l.c.d.palette[c]
	}
	l.c.r.Origin = l.c.origin
	l.c.r.FillRect(l.l, x, y, w, h, col)
}
