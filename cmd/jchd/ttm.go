package main

import (
	"fmt"
	"image"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"

	"github.com/xcifer/johnny-castaway-hd/internal/render"
	"github.com/xcifer/johnny-castaway-hd/internal/resource"
	"github.com/xcifer/johnny-castaway-hd/internal/script"
)

// loopPause is how long the empty island shows between two plays of a
// script, in redraws (one second).
const loopPause = 16

// ttmSpec is the script and clips chosen with -ttm.
type ttmSpec struct {
	name string
	ttm  *resource.TTM
	tags []uint16 // none: the whole script
}

// parseTTMSpec reads NAME or NAME:TAG,TAG,... such as "MJFISH:1,18".
func parseTTMSpec(d *gameData, arg string) (ttmSpec, error) {
	name, list, hasTags := strings.Cut(arg, ":")
	name = strings.ToUpper(name)
	if !strings.HasSuffix(name, ".TTM") {
		name += ".TTM"
	}
	e := d.archive.Find(name)
	if e == nil || e.Type() != "TTM" {
		var names []string
		for i := range d.archive.Entries {
			if e := &d.archive.Entries[i]; e.Type() == "TTM" {
				names = append(names, strings.TrimSuffix(e.Name, ".TTM"))
			}
		}
		slices.Sort(names)
		return ttmSpec{}, fmt.Errorf("no script %s; there are %s", name, strings.Join(names, " "))
	}
	ttm, err := resource.DecodeTTM(e)
	if err != nil {
		return ttmSpec{}, err
	}
	spec := ttmSpec{name: name, ttm: ttm}
	if !hasTags {
		return spec, nil
	}
	known := script.NewScript(name, ttm).Tags()
	for _, f := range strings.Split(list, ",") {
		n, err := strconv.ParseUint(strings.TrimSpace(f), 10, 16)
		if err != nil || !slices.Contains(known, uint16(n)) {
			return ttmSpec{}, fmt.Errorf("%s has no tag %q; its tags are %v", name, f, known)
		}
		spec.tags = append(spec.tags, uint16(n))
	}
	return spec, nil
}

// ttmScene plays clips of one script over the island, all starting
// together, then again after a pause, for checking them against the
// original. The clips that load pictures must be among them, as in the
// scene scripts: MJFISH:1,18 rather than MJFISH:18. Without tags it plays
// the whole script.
type ttmScene struct {
	r      *render.Renderer
	d      *gameData
	seed   uint64
	spec   ttmSpec
	at     image.Point // island offset
	sc     *script.Script
	island *islandScene
	sched  *script.Scheduler
	pause  int
	plays  int
}

func newTTMScene(r *render.Renderer, d *gameData, seed uint64, spec ttmSpec, at image.Point) *ttmScene {
	s := &ttmScene{r: r, d: d, seed: seed, spec: spec, at: at, sc: script.NewScript(spec.name, spec.ttm)}
	c := newScreenCanvas(r, d)
	c.screenLoaded = func() { s.island.still = true }
	s.sched = script.NewScheduler(c)
	s.start()
	return s
}

// start redraws the island, the same each time, and starts the clips.
func (s *ttmScene) start() {
	s.r.Saved.Clear()
	s.island = newIslandScene(s.r, s.d, rand.New(rand.NewPCG(s.seed, 0)), false, s.at)
	if len(s.spec.tags) == 0 {
		s.sched.AddScript(s.sc)
	}
	for _, tag := range s.spec.tags {
		s.sched.Add(s.sc, tag, 0)
	}
	s.plays++
}

// Tick advances the scene by one redraw.
func (s *ttmScene) Tick() {
	s.island.Tick()
	s.sched.Step()
	if len(s.sched.Threads()) == 0 {
		if s.pause == 0 {
			s.pause = loopPause
		}
		if s.pause--; s.pause > 0 {
			return
		}
		s.start()
	}
	s.sched.Play()
}

// Status describes what is playing, for the debug overlay.
func (s *ttmScene) Status() string {
	var b strings.Builder
	fmt.Fprintf(&b, "play %d", s.plays)
	for _, t := range s.sched.Threads() {
		fmt.Fprintf(&b, "\n  %v", t)
	}
	return b.String()
}
