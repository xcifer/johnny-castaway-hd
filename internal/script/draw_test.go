package script

import (
	"image"
	"math/rand/v2"
	"slices"
	"testing"
)

// pixels is a Layer that records the pixels filled in it.
type pixels map[image.Point]uint8

func (p pixels) Clear()                            { clear(p) }
func (p pixels) SetVisible(bool)                   {}
func (p pixels) SetClip(x1, y1, x2, y2 int)        {}
func (p pixels) ResetClip()                        {}
func (p pixels) DrawSprite(Sprite, int, int, bool) {}

func (p pixels) FillRect(x, y, w, h int, c uint8) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			p[image.Pt(px, py)] = c
		}
	}
}

func TestLinePixels(t *testing.T) {
	for _, c := range []struct {
		x1, y1, x2, y2 int
		want           []image.Point
	}{
		{0, 0, 4, 2, []image.Point{{0, 0}, {1, 0}, {2, 1}, {3, 1}}},
		{0, 0, 3, 1, []image.Point{{0, 0}, {1, 0}, {2, 1}}},
		{2, 3, 1, 0, []image.Point{{2, 3}, {2, 2}, {1, 1}}},
	} {
		var got []image.Point
		linePixels(c.x1, c.y1, c.x2, c.y2, func(x, y int) { got = append(got, image.Pt(x, y)) })
		if !slices.Equal(got, c.want) {
			t.Errorf("%d,%d-%d,%d: got %v, want %v", c.x1, c.y1, c.x2, c.y2, got, c.want)
		}
	}

	n := 0
	linePixels(5, 5, 5, 5, func(int, int) { n++ })
	if n != 0 {
		t.Errorf("zero-length line drew %d pixels", n)
	}
}

// Merging pixels into runs must not change which pixels are drawn.
func TestDrawLineMatchesPixels(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 2000 {
		x1, y1, x2, y2 := rng.IntN(80)-10, rng.IntN(80)-10, rng.IntN(80)-10, rng.IntN(80)-10
		want := pixels{}
		linePixels(x1, y1, x2, y2, func(x, y int) { want[image.Pt(x, y)] = 7 })
		got := pixels{}
		drawLine(got, x1, y1, x2, y2, 7)
		if len(got) != len(want) {
			t.Fatalf("line %d,%d-%d,%d: %d pixels, want %d", x1, y1, x2, y2, len(got), len(want))
		}
		for pt := range want {
			if got[pt] != 7 {
				t.Fatalf("line %d,%d-%d,%d: %v missing", x1, y1, x2, y2, pt)
			}
		}
	}
}

func TestDrawCircle(t *testing.T) {
	p := pixels{}
	if !drawCircle(p, 10, 20, 4, 4, 3, 3) {
		t.Fatal("4x4 circle refused")
	}
	rows := map[int][2]int{20: {11, 12}, 21: {10, 13}, 22: {10, 13}, 23: {11, 12}}
	for pt := range p {
		r, ok := rows[pt.Y]
		if !ok || pt.X < r[0] || pt.X > r[1] {
			t.Errorf("pixel %v outside the expected disc", pt)
		}
	}
	if len(p) != 12 {
		t.Errorf("%d pixels, want 12", len(p))
	}

	for _, size := range [][2]int{{4, 6}, {5, 5}, {0, 0}} {
		if drawCircle(pixels{}, 0, 0, size[0], size[1], 1, 1) {
			t.Errorf("%dx%d drawn, want refused", size[0], size[1])
		}
	}
}

// Every even circle stays in its box, touches all four sides, is mirror
// symmetric, and has its outline (fg) only on the edge of the disc.
func TestDrawCircleShape(t *testing.T) {
	for w := 2; w <= 120; w += 2 {
		p := pixels{}
		drawCircle(p, 0, 0, w, w, 1, 2)
		var hit [4]bool
		for pt, c := range p {
			if pt.X < 0 || pt.Y < 0 || pt.X >= w || pt.Y >= w {
				t.Fatalf("w=%d: %v outside the box", w, pt)
			}
			for _, m := range []image.Point{{w - 1 - pt.X, pt.Y}, {pt.X, w - 1 - pt.Y}} {
				if _, ok := p[m]; !ok {
					t.Fatalf("w=%d: %v has no mirror %v", w, pt, m)
				}
			}
			hit[0] = hit[0] || pt.X == 0
			hit[1] = hit[1] || pt.Y == 0
			hit[2] = hit[2] || pt.X == w-1
			hit[3] = hit[3] || pt.Y == w-1
			if c == 2 {
				for _, d := range []image.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if _, ok := p[pt.Add(d)]; !ok {
						t.Fatalf("w=%d: inside pixel %v on the edge", w, pt)
					}
				}
			}
		}
		if hit != [4]bool{true, true, true, true} {
			t.Errorf("w=%d: doesn't touch every side of its box", w)
		}
	}
}
