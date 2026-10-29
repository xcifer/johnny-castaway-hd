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
	"github.com/xcifer/johnny-castaway-hd/internal/story"
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

// adsSpec is the scene chosen with -ads.
type adsSpec struct {
	name string
	ads  *resource.ADS
	tag  uint16
}

// scriptNames lists the resources of one type, without the extension.
func scriptNames(d *gameData, typ string) string {
	var names []string
	for i := range d.archive.Entries {
		if e := &d.archive.Entries[i]; e.Type() == typ {
			names = append(names, strings.TrimSuffix(e.Name, "."+typ))
		}
	}
	slices.Sort(names)
	return strings.Join(names, " ")
}

// findScript finds NAME or NAME.TYP in the archive.
func findScript(d *gameData, name, typ string) (*resource.Entry, string, error) {
	name = strings.ToUpper(name)
	if !strings.HasSuffix(name, "."+typ) {
		name += "." + typ
	}
	e := d.archive.Find(name)
	if e == nil || e.Type() != typ {
		return nil, "", fmt.Errorf("no script %s; there are %s", name, scriptNames(d, typ))
	}
	return e, name, nil
}

// parseTTMSpec reads NAME or NAME:TAG,TAG,... such as "MJFISH:1,18".
func parseTTMSpec(d *gameData, arg string) (ttmSpec, error) {
	name, list, hasTags := strings.Cut(arg, ":")
	e, name, err := findScript(d, name, "TTM")
	if err != nil {
		return ttmSpec{}, err
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

// parseADSSpec reads NAME:SCENE, such as "FISHING:8".
func parseADSSpec(d *gameData, arg string) (adsSpec, error) {
	name, tag, _ := strings.Cut(arg, ":")
	e, name, err := findScript(d, name, "ADS")
	if err != nil {
		return adsSpec{}, err
	}
	ads, err := resource.DecodeADS(e)
	if err != nil {
		return adsSpec{}, err
	}
	var scenes []string
	for _, t := range ads.Tags {
		scenes = append(scenes, fmt.Sprintf("%d %s", t.ID, strings.TrimSpace(t.Name)))
		if strconv.Itoa(int(t.ID)) == strings.TrimSpace(tag) {
			return adsSpec{name: name, ads: ads, tag: t.ID}, nil
		}
	}
	return adsSpec{}, fmt.Errorf("%s has no scene %q; its scenes are:\n  %s", name, tag, strings.Join(scenes, "\n  "))
}

// loadScripts loads the animation scripts a scene script uses, by slot.
func loadScripts(d *gameData, ads *resource.ADS) map[uint16]*script.Script {
	scripts := map[uint16]*script.Script{}
	for _, res := range ads.Resources {
		e := d.archive.Find(res.Name)
		if e == nil {
			continue
		}
		if ttm, err := resource.DecodeTTM(e); err == nil {
			scripts[res.ID] = script.NewScript(res.Name, ttm)
		}
	}
	return scripts
}

// scriptScene plays clips of one script (-ttm) or one scene of a scene
// script (-ads) over the island, then again after a pause, for checking
// them against the original. With -ttm, the clips that load pictures must
// be among the tags, as in the scene scripts: MJFISH:1,18 rather than
// MJFISH:18. Without tags it plays the whole script.
type scriptScene struct {
	r      *render.Renderer
	d      *gameData
	seed   uint64
	at     image.Point // island offset
	ttm    *ttmSpec
	ads    *adsSpec
	sc     *script.Script
	island *island
	sched  *script.Scheduler
	scene  *script.Scene
	pause  int
	plays  int
}

func newScriptScene(r *render.Renderer, d *gameData, seed uint64, ttm *ttmSpec, ads *adsSpec, at image.Point) *scriptScene {
	s := &scriptScene{r: r, d: d, seed: seed, at: at, ttm: ttm, ads: ads}
	if ttm != nil {
		s.sc = script.NewScript(ttm.name, ttm.ttm)
	}
	c := newScreenCanvas(r, d)
	c.screenLoaded = func() { s.island.still = true }
	s.sched = script.NewScheduler(c)
	s.sched.Rand = rand.New(rand.NewPCG(seed, 1))
	s.start()
	return s
}

// start redraws the island, the same each time, and starts the clips or
// the scene.
func (s *scriptScene) start() {
	s.island = newIsland(s.r, s.d, rand.New(rand.NewPCG(s.seed, 0)), story.Island{At: s.at})
	s.plays++
	if s.ads != nil {
		// Each play loads the scene's scripts afresh, as the original
		// does for every scene.
		// The tag was checked when parsing the option.
		s.scene, _ = script.NewScene(s.sched, s.ads.name, s.ads.ads, loadScripts(s.d, s.ads.ads), s.ads.tag)
		return
	}
	if len(s.ttm.tags) == 0 {
		s.sched.AddScript(s.sc)
	}
	for _, tag := range s.ttm.tags {
		s.sched.Add(s.sc, tag, 0)
	}
}

// Tick advances the scene by one redraw.
func (s *scriptScene) Tick() {
	s.island.Tick()
	if s.scene != nil {
		s.scene.Step()
	} else {
		s.sched.Step()
	}
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
func (s *scriptScene) Status() string {
	var b strings.Builder
	fmt.Fprintf(&b, "play %d", s.plays)
	for _, t := range s.sched.Threads() {
		fmt.Fprintf(&b, "\n  %v", t)
	}
	return b.String()
}
