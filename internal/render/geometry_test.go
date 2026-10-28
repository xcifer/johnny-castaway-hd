package render

import (
	"image"
	"testing"
)

func TestNewFrame(t *testing.T) {
	for _, tc := range []struct {
		name          string
		w, h          int
		rect          image.Rectangle
		scale, offX   float64
		gameX0, gameX int // frame columns where logical 0 and 640 land
	}{
		{"1080p", 1920, 1080, image.Rect(0, 0, 1920, 1080), 2.25, 240, 240, 1680},
		{"4K", 3840, 2160, image.Rect(0, 0, 3840, 2160), 4.5, 480, 480, 3360},
		{"1440p", 2560, 1440, image.Rect(0, 0, 2560, 1440), 3, 320, 320, 2240},
		{"16:10 bars top and bottom", 1920, 1200, image.Rect(0, 60, 1920, 1140), 2.25, 240, 240, 1680},
		{"ultrawide bars left and right", 3440, 1440, image.Rect(440, 0, 3000, 1440), 3, 320, 320, 2240},
		{"4:3 screen", 1024, 768, image.Rect(0, 96, 1024, 672), 1.2, 128, 128, 896},
	} {
		f := NewFrame(tc.w, tc.h)
		if f.Rect != tc.rect || f.Scale != tc.scale || f.OffX != tc.offX {
			t.Errorf("%s: got rect %v scale %v offX %v, want %v %v %v", tc.name, f.Rect, f.Scale, f.OffX, tc.rect, tc.scale, tc.offX)
		}
		if got := f.X(0); got != tc.gameX0 {
			t.Errorf("%s: X(0) = %d, want %d", tc.name, got, tc.gameX0)
		}
		if got := f.X(640); got != tc.gameX {
			t.Errorf("%s: X(640) = %d, want %d", tc.name, got, tc.gameX)
		}
		w, h := f.Size()
		if w*9 != h*16 && (w*9-h*16 > 9 || h*16-w*9 > 16) {
			t.Errorf("%s: frame %dx%d is not 16:9", tc.name, w, h)
		}
		if f.Y(480) != h {
			t.Errorf("%s: Y(480) = %d, want frame height %d", tc.name, f.Y(480), h)
		}
	}
}

// Logical pixels must tile the game area without gaps or overlaps.
func TestLogicalPixelsTile(t *testing.T) {
	for _, size := range [][2]int{{1920, 1080}, {3840, 2160}, {1280, 720}, {1366, 768}} {
		f := NewFrame(size[0], size[1])
		for x := range LogicalWidth {
			w := f.X(x+1) - f.X(x)
			if w < int(f.Scale) || w > int(f.Scale)+1 {
				t.Fatalf("%v: logical column %d is %d px wide at scale %v", size, x, w, f.Scale)
			}
		}
		if r := f.LogicalRect(0, 0, LogicalWidth, LogicalHeight); r.Dx() != int(f.Scale*LogicalWidth+0.5) {
			t.Errorf("%v: game area %v", size, r)
		}
	}
}

func TestClip(t *testing.T) {
	f := NewFrame(1920, 1080)
	for _, tc := range []struct {
		name           string
		x1, y1, x2, y2 int
		want           image.Rectangle
	}{
		{"full zone reaches frame edges", 0, 0, 640, 480, image.Rect(0, 0, 1920, 1080)},
		{"639 counts as the right edge", 0, 0, 639, 479, image.Rect(0, 0, 1920, 1080)},
		{"inner zone", 100, 50, 200, 150, image.Rect(465, 113, 690, 338)},
		{"left edge only", 0, 50, 200, 150, image.Rect(0, 113, 690, 338)},
		{"negative start", -20, -5, 100, 100, image.Rect(0, 0, 465, 225)},
	} {
		if got := f.Clip(tc.x1, tc.y1, tc.x2, tc.y2); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
