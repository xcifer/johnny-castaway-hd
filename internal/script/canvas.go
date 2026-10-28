// Package script runs the game's TTM animation scripts. Scripts draw in the
// original 640x480 coordinates on a Canvas, which decides how that reaches
// the screen.
package script

// Sprite is a picture loaded by a Canvas. Scripts only hand it back to the
// Canvas that made it.
type Sprite any

// Canvas is what scripts draw on: a background picture, a layer of saved
// zones over it, and one layer per thread slot over that, higher slots on
// top.
type Canvas interface {
	// LoadBitmap returns the frames of a BMP resource.
	LoadBitmap(name string) ([]Sprite, error)
	// LoadScreen replaces the background with an SCR resource and clears
	// the saved zones.
	LoadScreen(name string) error
	// Layer returns the layer of thread slot i, 0 <= i < MaxThreads.
	Layer(i int) Layer
	// SaveZone copies the opaque pixels of a rectangle of l into the saved
	// zones, where they stay when l is cleared.
	SaveZone(l Layer, x, y, w, h int)
	// ClearSaved removes all saved zones.
	ClearSaved()
}

// Layer is a transparent surface the size of the screen. Color 0 is the
// transparent color: filling with it erases.
type Layer interface {
	// Clear makes the whole layer transparent, ignoring the clip zone.
	Clear()
	SetVisible(bool)
	// SetClip limits drawing to a zone; x2 and y2 are exclusive.
	SetClip(x1, y1, x2, y2 int)
	ResetClip()
	DrawSprite(s Sprite, x, y int, flip bool)
	FillRect(x, y, w, h int, c uint8)
}
