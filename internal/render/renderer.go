package render

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// Filter is how original-resolution art is scaled up.
type Filter int

const (
	SharpBilinear Filter = iota // crisp, even pixels (default)
	Nearest                     // blocky; pixel widths alternate at non-integer scales
	Smooth                      // bilinear; soft
	numFilters
)

func (f Filter) String() string {
	return [...]string{"sharp bilinear", "nearest", "smooth"}[f]
}

// Next returns the next filter, wrapping around.
func (f Filter) Next() Filter { return (f + 1) % numFilters }

//go:embed sharp.kage
var sharpSource []byte

// Sprite is a picture ready to draw: original pixels with real alpha.
type Sprite struct {
	img  *ebiten.Image
	W, H int
}

// NewSprite converts a decoded image using pal. If keyed, palette index 0
// is transparent (sprites); otherwise every pixel is opaque (backgrounds).
func NewSprite(src *resource.Image, pal color.Palette, keyed bool) *Sprite {
	rgba := image.NewNRGBA(image.Rect(0, 0, src.Width, src.Height))
	for i, idx := range src.Pix {
		if keyed && idx == 0 {
			continue
		}
		c := color.NRGBAModel.Convert(pal[idx]).(color.NRGBA)
		c.A = 0xFF
		rgba.Pix[4*i+0], rgba.Pix[4*i+1], rgba.Pix[4*i+2], rgba.Pix[4*i+3] = c.R, c.G, c.B, c.A
	}
	return &Sprite{img: ebiten.NewImageFromImage(rgba), W: src.Width, H: src.Height}
}

// NewBackground converts a full-screen picture for SetBackground. The last
// column of the ocean pictures breaks their dithering on up to 66 rows; at
// the edge of a 4:3 screen nobody saw it, but in widescreen it would show as
// a dark line where the picture meets its mirror image. It is replaced by
// the column two to the left, which has the same dither phase. Pictures
// without the flaw are unchanged.
func NewBackground(src *resource.Image, pal color.Palette) *Sprite {
	img := *src
	img.Pix = append([]uint8(nil), src.Pix...)
	if w := img.Width; w >= 3 {
		for y := range img.Height {
			img.Pix[y*w+w-1] = img.Pix[y*w+w-3]
		}
	}
	return NewSprite(&img, pal, false)
}

// Layer is a transparent, frame-sized image that the game draws into. Its
// image is only allocated when something is drawn, so unused layers cost
// no memory.
type Layer struct {
	img    *ebiten.Image
	w, h   int
	clip   image.Rectangle
	hidden bool
}

func (l *Layer) image() *ebiten.Image {
	if l.img == nil {
		l.img = ebiten.NewImage(l.w, l.h)
	}
	return l.img
}

// Clear makes the whole layer transparent.
func (l *Layer) Clear() {
	if l.img != nil {
		l.img.Clear()
	}
}

// SetVisible shows or hides the layer. Layers start visible.
func (l *Layer) SetVisible(v bool) { l.hidden = !v }

// target returns the part of the layer inside the clip zone.
func (l *Layer) target() *ebiten.Image {
	return l.image().SubImage(l.clip).(*ebiten.Image)
}

// Renderer owns the frame and its layers. They are drawn back to front:
// background, saved zones, then the other layers in the order they were
// created. Coordinates are logical (640x480) plus Origin, the offset the
// game uses to place the island.
type Renderer struct {
	Frame      Frame
	Filter     Filter
	Origin     image.Point
	Background *Layer
	Saved      *Layer
	layers     []*Layer
	sharp      *ebiten.Shader
}

// New creates a renderer for a screen of the given size in pixels.
func New(screenW, screenH int) (*Renderer, error) {
	sharp, err := ebiten.NewShader(sharpSource)
	if err != nil {
		return nil, fmt.Errorf("sharp bilinear shader: %w", err)
	}
	r := &Renderer{Frame: NewFrame(screenW, screenH), sharp: sharp}
	r.Background = r.newLayer()
	r.Saved = r.newLayer()
	return r, nil
}

func (r *Renderer) newLayer() *Layer {
	w, h := r.Frame.Size()
	return &Layer{w: w, h: h, clip: r.Frame.Full()}
}

// NewLayer adds a layer on top of the existing ones.
func (r *Renderer) NewLayer() *Layer {
	l := r.newLayer()
	r.layers = append(r.layers, l)
	return l
}

// SetClip limits drawing on l to a logical zone (x2, y2 exclusive). Zones
// touching the edges of the 640x480 area reach the edges of the frame.
func (r *Renderer) SetClip(l *Layer, x1, y1, x2, y2 int) {
	o := r.Origin
	l.clip = r.Frame.Clip(x1+o.X, y1+o.Y, x2+o.X, y2+o.Y)
}

// ResetClip lets l be drawn on everywhere.
func (r *Renderer) ResetClip(l *Layer) { l.clip = r.Frame.Full() }

// DrawSprite draws s with its top-left corner at logical (x, y), mirrored
// horizontally if flip is set.
func (r *Renderer) DrawSprite(l *Layer, s *Sprite, x, y int, flip bool) {
	x += r.Origin.X
	y += r.Origin.Y
	r.drawScaled(l.target(), s, float64(r.Frame.X(x)), float64(r.Frame.Y(y)), flip)
}

// drawScaled draws s at frame position (fx, fy), scaled by Frame.Scale.
func (r *Renderer) drawScaled(dst *ebiten.Image, s *Sprite, fx, fy float64, flip bool) {
	scale := r.Frame.Scale
	var geo ebiten.GeoM
	if flip {
		geo.Scale(-1, 1)
		geo.Translate(float64(s.W), 0)
	}
	geo.Scale(scale, scale)
	geo.Translate(fx, fy)

	if r.Filter == SharpBilinear {
		op := &ebiten.DrawRectShaderOptions{GeoM: geo}
		op.Images[0] = s.img
		op.Uniforms = map[string]any{"Scale": float32(scale)}
		dst.DrawRectShader(s.W, s.H, r.sharp, op)
		return
	}
	op := &ebiten.DrawImageOptions{GeoM: geo, Filter: ebiten.FilterNearest}
	if r.Filter == Smooth {
		op.Filter = ebiten.FilterLinear
	}
	dst.DrawImage(s.img, op)
}

// FillRect fills a logical rectangle with c.
func (r *Renderer) FillRect(l *Layer, x, y, w, h int, c color.Color) {
	rect := r.Frame.LogicalRect(x+r.Origin.X, y+r.Origin.Y, w, h).Intersect(l.clip)
	if !rect.Empty() {
		l.image().SubImage(rect).(*ebiten.Image).Fill(c)
	}
}

// SetBackground draws a full-screen picture into the background layer. The
// strips beside the 640-wide game area show its mirror image, so the
// picture continues to the frame edges. The mirror is taken around the edge
// column without repeating it (column 640 shows 638, column -1 shows 1):
// repeating it would break the checkerboard dithering into a visible stripe.
func (r *Renderer) SetBackground(s *Sprite) {
	bg := r.Background.image()
	bg.Clear()
	r.drawScaled(bg, s, float64(r.Frame.X(1-s.W)), 0, true)
	r.drawScaled(bg, s, float64(r.Frame.X(s.W-1)), 0, true)
	r.drawScaled(bg, s, float64(r.Frame.X(0)), 0, false)
}

// CopyZoneToSaved copies the opaque pixels of a logical rectangle of l
// into the saved-zones layer, where they stay until it is cleared.
func (r *Renderer) CopyZoneToSaved(l *Layer, x, y, w, h int) {
	rect := r.Frame.LogicalRect(x+r.Origin.X, y+r.Origin.Y, w, h).Intersect(r.Frame.Full())
	if rect.Empty() || l.img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	r.Saved.image().DrawImage(l.img.SubImage(rect).(*ebiten.Image), op)
}

// Compose draws all visible layers onto screen, inside the frame, with
// black bars around it.
func (r *Renderer) Compose(screen *ebiten.Image) {
	screen.Fill(color.Black)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Frame.Rect.Min.X), float64(r.Frame.Rect.Min.Y))
	for _, l := range append([]*Layer{r.Background, r.Saved}, r.layers...) {
		if l.img != nil && !l.hidden {
			screen.DrawImage(l.img, op)
		}
	}
}
