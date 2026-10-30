package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/xcifer/johnny-castaway-hd/internal/render"
	"github.com/xcifer/johnny-castaway-hd/internal/resource"
	"github.com/xcifer/johnny-castaway-hd/internal/story"
)

// gameData is the original data: the archive scripts load from, and what
// the island and the walks need decoded.
type gameData struct {
	archive    *resource.Archive
	palette    color.Palette
	oceans     []*resource.Image
	night      *resource.Image
	background *resource.Sprites // BACKGRND.BMP: island, palm, clouds, waves
	raft       *resource.Sprites // MRAFT.BMP
	holiday    *resource.Sprites // HOLIDAY.BMP
	walk       *resource.Sprites // JOHNWALK.BMP
	walkTable  []resource.WalkStep
	sounds     []resource.Sound

	ready   map[*resource.Sprites][]*render.Sprite
	screens map[*resource.Image]*render.Sprite
}

func loadGameData(dir string) (*gameData, error) {
	a, err := resource.Load(dir)
	if err != nil {
		return nil, err
	}
	find := func(name string) (*resource.Entry, error) {
		if e := a.Find(name); e != nil {
			return e, nil
		}
		return nil, fmt.Errorf("%s not found", name)
	}

	d := &gameData{archive: a, ready: map[*resource.Sprites][]*render.Sprite{}, screens: map[*resource.Image]*render.Sprite{}}
	e, err := find("JOHNCAST.PAL")
	if err != nil {
		return nil, err
	}
	pal, err := resource.DecodePAL(e)
	if err != nil {
		return nil, err
	}
	for i := range 16 {
		d.palette = append(d.palette, pal.Color(i))
	}

	for _, name := range []string{"OCEAN00.SCR", "OCEAN01.SCR", "OCEAN02.SCR", "NIGHT.SCR"} {
		e, err := find(name)
		if err != nil {
			return nil, err
		}
		img, err := resource.DecodeSCR(e)
		if err != nil {
			return nil, err
		}
		if name == "NIGHT.SCR" {
			d.night = img
		} else {
			d.oceans = append(d.oceans, img)
		}
	}

	for name, dst := range map[string]**resource.Sprites{
		"BACKGRND.BMP": &d.background, "MRAFT.BMP": &d.raft, "HOLIDAY.BMP": &d.holiday, "JOHNWALK.BMP": &d.walk,
	} {
		e, err := find(name)
		if err != nil {
			return nil, err
		}
		if *dst, err = resource.DecodeBMP(e); err != nil {
			return nil, err
		}
	}

	exe, err := os.ReadFile(filepath.Join(dir, "SCRANTIC.SCR"))
	if err != nil {
		return nil, err
	}
	if d.walkTable, err = resource.WalkTable(exe); err != nil {
		return nil, err
	}
	if d.sounds, err = resource.Sounds(exe); err != nil {
		return nil, err
	}
	return d, nil
}

// sprites prepares decoded pictures for drawing, once.
func (d *gameData) sprites(s *resource.Sprites) []*render.Sprite {
	if out, ok := d.ready[s]; ok {
		return out
	}
	out := make([]*render.Sprite, len(s.Frames))
	for i := range s.Frames {
		out[i] = render.NewSprite(&s.Frames[i], d.palette, true)
	}
	d.ready[s] = out
	return out
}

// screen prepares a full-screen picture for drawing, once.
func (d *gameData) screen(img *resource.Image) *render.Sprite {
	s, ok := d.screens[img]
	if !ok {
		s = render.NewBackground(img, d.palette)
		d.screens[img] = s
	}
	return s
}

// Timing in redraws of 62.5 ms, measured in recordings (docs/timing.md): one
// shore wave moves on every redraw, and walking takes one step per redraw
// and a pause of about 13 after arriving.
const (
	waveDelay   = 1
	walkDelay   = 1
	arriveDelay = 13
)

// leftIsland is where some scenes move the island (jc_reborn's story.c):
// 272 pixels to the left. Their scripts are drawn for that position.
var leftIsland = image.Pt(-272, 0)

// Where the shore waves are drawn, and their first picture in BACKGRND.BMP;
// each has three. At low tide the rock has waves too.
var (
	highTideWaves = [][3]int{{270, 306, 3}, {364, 319, 6}, {518, 303, 9}}
	lowTideWaves  = [][3]int{{129, 340, 39}, {233, 323, 30}, {367, 356, 33}, {558, 323, 36}}
)

// island draws the island as the story sets it up, and keeps its waves
// moving. Layout follows jc_reborn's island.c.
type island struct {
	r     *render.Renderer
	bg    []*render.Sprite
	state story.Island
	still bool // the waves stop when a script replaces the background

	waveSlot, waveFrame int // which shore wave to redraw, and its frame
	nextWave, tick      int
}

func newIsland(r *render.Renderer, d *gameData, rng *rand.Rand, state story.Island) *island {
	is := &island{r: r, bg: d.sprites(d.background), state: state}
	bgl := r.Background
	r.Saved.Clear()

	sky := d.night
	if !state.Night {
		sky = d.oceans[rng.IntN(len(d.oceans))]
	}
	r.SetBackground(d.screen(sky))

	if state.Raft > 0 {
		x, y := 512, 266
		if state.LowTide {
			x, y = 529, 281
		}
		is.draw(bgl, d.sprites(d.raft)[state.Raft-1], x, y)
	}

	// Clouds, sprites 15-17, don't move with the island. Most often there is
	// one or none.
	clouds := 5
	switch {
	case rng.IntN(2) == 0:
		clouds = 1
	case rng.IntN(2) == 0:
		clouds = 0
	case rng.IntN(4) != 0:
		clouds = 2
	case rng.IntN(4) != 0:
		clouds = 3
	case rng.IntN(4) != 0:
		clouds = 4
	}
	flip := rng.IntN(2) == 0 // the way the wind blows
	r.Origin = image.Point{}
	for range clouds {
		n := rng.IntN(3)
		size := [3][2]int{{129, 36}, {192, 57}, {264, 76}}[n]
		r.DrawSprite(bgl, is.bg[15+n], rng.IntN(640-size[0]), rng.IntN(135-size[1]), flip)
	}

	is.draw(bgl, is.bg[0], 288, 279)  // island
	is.draw(bgl, is.bg[13], 442, 148) // trunk
	is.draw(bgl, is.bg[12], 365, 122) // leaves
	is.draw(bgl, is.bg[14], 396, 279) // palm shadow
	if state.LowTide {
		is.draw(bgl, is.bg[1], 249, 303) // shore
		is.draw(bgl, is.bg[2], 150, 328) // rock
	}
	for range 4 {
		is.animateWaves()
	}
	return is
}

// draw draws a sprite at x, y on the island.
func (is *island) draw(l *render.Layer, s *render.Sprite, x, y int) {
	is.r.Origin = is.state.At
	is.r.DrawSprite(l, s, x, y, false)
}

// drawHoliday decorates the island on l for the holiday, if there is one.
func (is *island) drawHoliday(l *render.Layer, d *gameData) {
	l.Clear()
	if is.state.Holiday == story.NoHolidayToday {
		return
	}
	pos := map[story.Holiday][2]int{
		story.Halloween:     {410, 298},
		story.StPatricksDay: {333, 286},
		story.Christmas:     {404, 267},
		story.NewYear:       {361, 155},
	}[is.state.Holiday]
	is.draw(l, d.sprites(d.holiday)[int(is.state.Holiday)-1], pos[0], pos[1])
}

// drawTree draws the palm tree on l, in front of what is behind it.
func (is *island) drawTree(l *render.Layer) {
	is.draw(l, is.bg[13], 442, 148)
	is.draw(l, is.bg[12], 365, 122)
}

// animateWaves redraws one of the shore waves.
func (is *island) animateWaves() {
	waves := highTideWaves
	if is.state.LowTide {
		waves = lowTideWaves
	}
	is.waveSlot = (is.waveSlot + 1) % len(waves)
	w := waves[is.waveSlot]
	is.draw(is.r.Background, is.bg[w[2]+is.waveFrame], w[0], w[1])
	if is.waveSlot == 0 {
		is.waveFrame = (is.waveFrame + 1) % 3
	}
}

// Tick advances the waves by one redraw.
func (is *island) Tick() {
	is.tick++
	if is.tick >= is.nextWave && !is.still {
		is.animateWaves()
		is.nextWave = is.tick + waveDelay
	}
}
