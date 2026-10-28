// Package render draws the game into a 16:9 frame. Game logic works in the
// original 640x480 coordinates; this package maps them to screen pixels.
package render

import (
	"image"
	"math"
)

// The original game's screen.
const (
	LogicalWidth  = 640
	LogicalHeight = 480
)

// Frame is the 16:9 picture, fitted into the screen. The 640x480 game area
// is scaled to the frame's height and centered; the strips left and right
// of it show extended background. At 1920x1080 the scale is 2.25 and the
// game area starts at x = 240.
type Frame struct {
	Screen image.Point     // size of the whole screen in pixels
	Rect   image.Rectangle // the frame within the screen; black bars around it
	Scale  float64         // screen pixels per logical pixel
	OffX   float64         // left edge of the game area, in frame pixels
}

// NewFrame fits a 16:9 frame into a screen of the given size.
func NewFrame(screenW, screenH int) Frame {
	w, h := screenW, screenH
	if w*9 > h*16 {
		w = int(math.Round(float64(h) * 16 / 9))
	} else {
		h = int(math.Round(float64(w) * 9 / 16))
	}
	x := (screenW - w) / 2
	y := (screenH - h) / 2
	scale := float64(h) / LogicalHeight
	return Frame{
		Screen: image.Pt(screenW, screenH),
		Rect:   image.Rect(x, y, x+w, y+h),
		Scale:  scale,
		OffX:   (float64(w) - LogicalWidth*scale) / 2,
	}
}

// Size returns the frame's width and height.
func (f Frame) Size() (int, int) {
	return f.Rect.Dx(), f.Rect.Dy()
}

// X returns the frame pixel column where logical column x starts.
func (f Frame) X(x int) int {
	return int(math.Round(f.OffX + float64(x)*f.Scale))
}

// Y returns the frame pixel row where logical row y starts.
func (f Frame) Y(y int) int {
	return int(math.Round(float64(y) * f.Scale))
}

// LogicalRect returns the frame pixels covered by a logical rectangle.
func (f Frame) LogicalRect(x, y, w, h int) image.Rectangle {
	return image.Rect(f.X(x), f.Y(y), f.X(x+w), f.Y(y+h))
}

// Clip converts a logical clip zone (x2, y2 exclusive) to frame pixels. A
// zone that reaches the left, top, right or bottom edge of the 640x480 area
// extends to the edge of the frame, so drawing continues into the
// widescreen strips.
func (f Frame) Clip(x1, y1, x2, y2 int) image.Rectangle {
	w, h := f.Size()
	r := image.Rect(f.X(x1), f.Y(y1), f.X(x2), f.Y(y2))
	if x1 <= 0 {
		r.Min.X = 0
	}
	if y1 <= 0 {
		r.Min.Y = 0
	}
	if x2 >= LogicalWidth-1 {
		r.Max.X = w
	}
	if y2 >= LogicalHeight-1 {
		r.Max.Y = h
	}
	return r.Intersect(image.Rect(0, 0, w, h))
}

// Full returns the whole frame as a clip rectangle.
func (f Frame) Full() image.Rectangle {
	w, h := f.Size()
	return image.Rect(0, 0, w, h)
}
