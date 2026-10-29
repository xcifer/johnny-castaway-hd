package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/xcifer/johnny-castaway-hd/internal/render"
	"github.com/xcifer/johnny-castaway-hd/internal/resource"
	"github.com/xcifer/johnny-castaway-hd/internal/script"
	"github.com/xcifer/johnny-castaway-hd/internal/story"
)

// introDelay is how long the title picture shows, in redraws: jc_reborn's
// 100 ticks, not checked against the original.
const introDelay = 100 / script.FrameTicks

// storyScene is the screensaver itself: run after run of scenes, with
// Johnny walking from each to the next.
type storyScene struct {
	r      *render.Renderer
	d      *gameData
	rng    *rand.Rand
	now    func() time.Time
	day    func() int // the day of the story, asked at the start of each run
	canvas *screenCanvas
	sched  *script.Scheduler

	walker, holiday *render.Layer
	walkSprites     []*render.Sprite

	island *island // nil while a run plays off the island
	run    *story.Run
	runDay int
	runs   int
	next   int // index in run.Scenes of the scene to play next

	scene   *script.Scene // the scene playing, or nil
	playing *story.Scene

	// Where Johnny ended the last scene, if he is to walk to the next.
	from       story.Spot
	fromFacing story.Heading
	mustWalk   bool

	walk []story.WalkFrame // the walk in progress, or nil
	step int
	wait int // redraws until the next picture of the walk, or of the intro
}

func newStoryScene(r *render.Renderer, d *gameData, rng *rand.Rand, now func() time.Time, day func() int) *storyScene {
	s := &storyScene{r: r, d: d, rng: rng, now: now, day: day, walkSprites: d.sprites(d.walk)}
	s.canvas = newScreenCanvas(r, d)
	s.canvas.screenLoaded = func() {
		if s.island != nil {
			s.island.still = true
		}
	}
	s.walker = r.NewLayer()
	s.holiday = r.NewLayer() // on top of everything, as in the original
	s.sched = script.NewScheduler(s.canvas)
	s.sched.Rand = rng

	if err := s.canvas.LoadScreen("INTRO.SCR"); err == nil {
		s.wait = introDelay
	}
	return s
}

// Tick advances the story by one redraw.
func (s *storyScene) Tick() {
	if s.island != nil {
		s.island.Tick()
	}
	switch {
	case s.walk != nil:
		s.tickWalk()
		return
	case s.scene != nil:
		s.scene.Step()
		if !s.scene.Done() {
			break
		}
		s.scene = nil
		s.from, s.fromFacing = s.playing.End, s.playing.EndFacing
		s.mustWalk = s.island != nil && s.playing.Flags&story.Final == 0
		s.next++
		fallthrough
	default:
		if s.wait > 0 {
			s.wait--
			break
		}
		s.startNext()
	}
	s.sched.Play()
}

// startNext starts a walk, or else the next scene, and a new run when the
// last one is over.
func (s *storyScene) startNext() {
	if s.run == nil || s.next == len(s.run.Scenes) {
		s.newRun()
	}
	sc := s.run.Scenes[s.next]
	if s.mustWalk {
		s.mustWalk = false
		s.walk = story.Walk(s.d.walkTable, s.rng, s.from, s.fromFacing, sc.Start, sc.StartFacing)
		s.step = 0
		s.drawWalker()
		return
	}

	s.playing = sc
	s.canvas.origin = s.run.Offset(sc)
	ads, err := s.loadADS(sc.ADS)
	if err == nil {
		s.scene, err = script.NewScene(s.sched, sc.ADS, ads, loadScripts(s.d, ads), sc.Tag)
	}
	if err != nil {
		// Skip a scene that can't be played. The tests check every scene
		// of the story against the data, so only a broken install gets here.
		log.Printf("story: %s %d: %v", sc.ADS, sc.Tag, err)
		s.next++
	}
}

func (s *storyScene) loadADS(name string) (*resource.ADS, error) {
	e := s.d.archive.Find(name)
	if e == nil {
		return nil, fmt.Errorf("%s not found", name)
	}
	return resource.DecodeADS(e)
}

// newRun plans the next run and sets up the island for it.
func (s *storyScene) newRun() {
	s.runDay = s.day()
	s.run = story.NewRun(s.rng, s.runDay, s.now())
	s.runs++
	s.next = 0
	s.mustWalk = false
	s.walker.Clear()
	final := s.run.Scenes[len(s.run.Scenes)-1]
	if final.Flags&story.OnIsland == 0 {
		// Its scripts bring their own pictures.
		s.island = nil
		s.r.Background.Clear()
		s.r.Saved.Clear()
		s.holiday.Clear()
		return
	}
	s.island = newIsland(s.r, s.d, s.rng, s.run.Island)
	s.island.drawHoliday(s.holiday, s.d)
}

func (s *storyScene) drawWalker() {
	f := s.walk[s.step]
	s.walker.Clear()
	s.r.Origin = s.run.Island.At
	s.r.DrawSprite(s.walker, s.walkSprites[f.Sprite], f.X-1, f.Y, f.Flip)
	if f.BehindTree {
		s.island.drawTree(s.walker)
	}
	s.wait = walkDelay
	if s.step == len(s.walk)-1 {
		s.wait = arriveDelay
	}
}

func (s *storyScene) tickWalk() {
	if s.wait--; s.wait > 0 {
		return
	}
	if s.step++; s.step < len(s.walk) {
		s.drawWalker()
		return
	}
	// Arrived: the scene starts in this redraw, as Johnny disappears.
	s.walk = nil
	s.walker.Clear()
	s.startNext()
	s.sched.Play()
}

// Status describes what is playing, for the debug overlay.
func (s *storyScene) Status() string {
	if s.run == nil {
		return "intro"
	}
	is := s.run.Island
	var b strings.Builder
	fmt.Fprintf(&b, "story day %d  run %d  scene %d/%d", s.runDay, s.runs, s.next+1, len(s.run.Scenes))
	if s.island != nil {
		fmt.Fprintf(&b, "\nisland at %d,%d  raft %d", is.At.X, is.At.Y, is.Raft)
		if is.LowTide {
			b.WriteString("  low tide")
		}
	} else {
		b.WriteString("\noff the island")
	}
	if is.Night {
		b.WriteString("  night")
	}
	if is.Holiday != story.NoHolidayToday {
		fmt.Fprintf(&b, "  %v", is.Holiday)
	}
	switch {
	case s.walk != nil:
		sc := s.run.Scenes[s.next]
		fmt.Fprintf(&b, "\nwalking %v to %v, step %d/%d", s.from, sc.Start, s.step+1, len(s.walk))
	case s.scene != nil:
		fmt.Fprintf(&b, "\n%s:%d", s.playing.ADS, s.playing.Tag)
		for _, t := range s.sched.Threads() {
			fmt.Fprintf(&b, "\n  %v", t)
		}
	}
	return b.String()
}
