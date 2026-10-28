package script

// Line and circle drawing, ported from grDrawLine and grDrawCircle in
// graphics.c of jc_reborn by Jeremie Guillaume (GPL-3.0), which aim to match
// the original pixel for pixel.

// runs draws pixels, merging neighbors in a row or column into one
// rectangle so a line costs a few fills instead of one per pixel.
type runs struct {
	l          Layer
	c          uint8
	x, y, w, h int
}

func (r *runs) add(x, y int) {
	switch {
	case r.w == 0:
	case r.h == 1 && y == r.y && x == r.x+r.w:
		r.w++
		return
	case r.h == 1 && y == r.y && x == r.x-1:
		r.x--
		r.w++
		return
	case r.w == 1 && x == r.x && y == r.y+r.h:
		r.h++
		return
	case r.w == 1 && x == r.x && y == r.y-1:
		r.y--
		r.h++
		return
	default:
		r.flush()
	}
	r.x, r.y, r.w, r.h = x, y, 1, 1
}

func (r *runs) flush() {
	if r.w > 0 {
		r.l.FillRect(r.x, r.y, r.w, r.h, r.c)
		r.w = 0
	}
}

// linePixels calls plot for each pixel of the line from (x1, y1) towards
// (x2, y2). The end point itself is left out.
func linePixels(x1, y1, x2, y2 int, plot func(x, y int)) {
	dx, dy := abs(x2-x1), abs(y2-y1)
	xinc, yinc := -1, -1
	if x2 > x1 {
		xinc = 1
	}
	if y2 > y1 {
		yinc = 1
	}
	x, y := x1, y1
	if dy < dx {
		cumul := (dx + 1) >> 1
		for range dx {
			plot(x, y)
			x += xinc
			cumul += dy
			if cumul > dx {
				cumul -= dx
				y += yinc
			}
		}
	} else {
		cumul := (dy + 1) >> 1
		for range dy {
			plot(x, y)
			y += yinc
			cumul += dx
			if cumul > dy {
				cumul -= dy
				x += xinc
			}
		}
	}
}

func drawLine(l Layer, x1, y1, x2, y2 int, c uint8) {
	r := runs{l: l, c: c}
	linePixels(x1, y1, x2, y2, r.add)
	r.flush()
}

// drawCircle draws the circle that fills the w x h box at (x, y): its inside
// in bg and, if fg differs, its outline in fg. Like the original data, it
// only handles circles with an even diameter and reports false otherwise.
func drawCircle(l Layer, x, y, w, h int, fg, bg uint8) bool {
	if w != h || w < 2 || w%2 != 0 {
		return false
	}
	r := w/2 - 1
	xc, yc := x+r, y+r
	span := func(x1, x2, y int) { l.FillRect(x1, y, x2-x1+1, 1, bg) }
	px, py, d := 0, r, 1-r
	for {
		span(xc-px, xc+px+1, yc+py+1)
		span(xc-px, xc+px+1, yc-py)
		span(xc-py, xc+py+1, yc+px+1)
		span(xc-py, xc+py+1, yc-px)
		if py-px <= 1 {
			break
		}
		if d < 0 {
			d += px<<1 + 3
		} else {
			d += (px-py)<<1 + 5
			py--
		}
		px++
	}
	if fg == bg {
		return true
	}
	dot := func(x, y int) { l.FillRect(x, y, 1, 1, fg) }
	px, py, d = 0, r, 1-r
	for {
		dot(xc-px, yc+py+1)
		dot(xc+px+1, yc+py+1)
		dot(xc-px, yc-py)
		dot(xc+px+1, yc-py)
		dot(xc-py, yc+px+1)
		dot(xc+py+1, yc+px+1)
		dot(xc-py, yc-px)
		dot(xc+py+1, yc-px)
		if py-px <= 1 {
			break
		}
		if d < 0 {
			d += px<<1 + 3
		} else {
			d += (px-py)<<1 + 5
			py--
		}
		px++
	}
	return true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
