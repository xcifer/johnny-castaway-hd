package main

import (
	"fmt"
	"image/color"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/xcifer/johnny-castaway-hd/internal/render"
	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// gameData is the original data: the archive scripts load from, and what
// the island scene needs decoded.
type gameData struct {
	archive    *resource.Archive
	palette    color.Palette
	oceans     []*resource.Image
	background *resource.Sprites // BACKGRND.BMP: island, palm, clouds, waves
	walk       *resource.Sprites // JOHNWALK.BMP
	walkPaths  [][]resource.WalkStep
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

	d := &gameData{archive: a}
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

	for i := range 3 {
		e, err := find(fmt.Sprintf("OCEAN0%d.SCR", i))
		if err != nil {
			return nil, err
		}
		img, err := resource.DecodeSCR(e)
		if err != nil {
			return nil, err
		}
		d.oceans = append(d.oceans, img)
	}

	for name, dst := range map[string]**resource.Sprites{"BACKGRND.BMP": &d.background, "JOHNWALK.BMP": &d.walk} {
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
	steps, err := resource.WalkTable(exe)
	if err != nil {
		return nil, err
	}
	d.walkPaths = resource.WalkPaths(steps)
	return d, nil
}

// Timing in ticks of 20 ms, from jc_reborn: shore waves advance every 8
// ticks, a walking step takes 6, and Johnny pauses 80 after arriving.
const (
	waveDelay   = 8
	walkDelay   = 6
	arriveDelay = 80
)

// islandScene shows the island at high tide with waves and, if walking,
// Johnny walking every path of the walk table in turn. Layout follows
// jc_reborn's island.c. It stands in for the scripted scenes until those
// are implemented.
type islandScene struct {
	r       *render.Renderer
	bg      []*render.Sprite
	walk    []*render.Sprite
	paths   [][]resource.WalkStep
	walkers *render.Layer // nil if Johnny doesn't walk
	still   bool          // the waves stop when the island is replaced

	waveSlot, waveFrame int // which shore wave to redraw, and its frame
	path, step          int
	nextWave, nextWalk  int
	tick                int
}

func newIslandScene(r *render.Renderer, d *gameData, rng *rand.Rand, walking bool) *islandScene {
	s := &islandScene{r: r, paths: d.walkPaths}
	for i := range d.background.Frames {
		s.bg = append(s.bg, render.NewSprite(&d.background.Frames[i], d.palette, true))
	}
	for i := range d.walk.Frames {
		s.walk = append(s.walk, render.NewSprite(&d.walk.Frames[i], d.palette, true))
	}

	ocean := d.oceans[rng.IntN(len(d.oceans))]
	r.SetBackground(render.NewBackground(ocean, d.palette))
	bgl := r.Background

	// Clouds: sprites 15-17, placed at random over the sky.
	flip := rng.IntN(2) == 0
	for range rng.IntN(4) {
		n := rng.IntN(3)
		size := [3][2]int{{129, 36}, {192, 57}, {264, 76}}[n]
		r.DrawSprite(bgl, s.bg[15+n], rng.IntN(640-size[0]), rng.IntN(135-size[1]), flip)
	}

	r.DrawSprite(bgl, s.bg[0], 288, 279, false)  // island
	r.DrawSprite(bgl, s.bg[13], 442, 148, false) // trunk
	r.DrawSprite(bgl, s.bg[12], 365, 122, false) // leaves
	r.DrawSprite(bgl, s.bg[14], 396, 279, false) // palm shadow
	for range 4 {
		s.animateWaves()
	}

	if walking {
		s.walkers = r.NewLayer()
		s.drawWalker()
	}
	return s
}

// animateWaves redraws one of the three high-tide shore waves.
func (s *islandScene) animateWaves() {
	s.waveSlot = (s.waveSlot + 1) % 3
	pos := [3][2]int{{270, 306}, {364, 319}, {518, 303}}[s.waveSlot]
	s.r.DrawSprite(s.r.Background, s.bg[3+3*s.waveSlot+s.waveFrame], pos[0], pos[1], false)
	if s.waveSlot == 0 {
		s.waveFrame = (s.waveFrame + 1) % 3
	}
}

func (s *islandScene) drawWalker() {
	st := s.paths[s.path][s.step]
	s.walkers.Clear()
	s.r.DrawSprite(s.walkers, s.walk[st.Sprite], st.X-1, st.Y, st.Flip)
}

// Tick advances the scene by one 20 ms tick.
func (s *islandScene) Tick() {
	s.tick++
	if s.tick >= s.nextWave && !s.still {
		s.animateWaves()
		s.nextWave = s.tick + waveDelay
	}
	if s.walkers != nil && s.tick >= s.nextWalk {
		s.nextWalk = s.tick + walkDelay
		s.step++
		if s.step == len(s.paths[s.path]) {
			s.step = 0
			s.path = (s.path + 1) % len(s.paths)
		}
		s.drawWalker()
		if s.step == len(s.paths[s.path])-1 {
			s.nextWalk = s.tick + arriveDelay
		}
	}
}

// Status describes what is playing, for the debug overlay.
func (s *islandScene) Status() string {
	if s.walkers == nil {
		return ""
	}
	return fmt.Sprintf("walk path %d/%d step %d/%d", s.path+1, len(s.paths), s.step+1, len(s.paths[s.path]))
}
