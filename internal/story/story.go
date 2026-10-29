// Package story decides what Johnny does: which scenes play, in what order,
// and how the island looks while they do. It follows story.c, story_data.h
// and island.c of jc_reborn by Jeremie Guillaume (GPL-3.0).
package story

import (
	"image"
	"math/rand/v2"
	"time"
)

// Spot is a place on the island Johnny walks between.
type Spot int

const (
	SpotA Spot = iota
	SpotB
	SpotC
	SpotD
	SpotE
	SpotF
)

func (s Spot) String() string { return string(rune('A' + s)) }

// Heading is the way Johnny faces, in eighths of a turn clockwise from south
// as seen on the screen.
type Heading int

const (
	South Heading = iota
	SouthWest
	West
	NorthWest
	North
	NorthEast
	East
	SouthEast
)

// Flags say where a scene can go in the story and what the island must be
// like for it.
type Flags uint8

const (
	Final      Flags = 1 << iota // can end a run of scenes
	First                        // can only start one
	OnIsland                     // plays on the island
	LeftIsland                   // drawn for the island moved 272 pixels left
	AnyPlace                     // works wherever the island is
	LowTideOK                    // works at low tide
	NoRaft                       // the raft isn't there
	NoHoliday                    // holiday decorations would show through
)

// Scene is one scene of a scene script and its place in the story.
type Scene struct {
	ADS         string
	Tag         uint16
	Start       Spot // where Johnny is when the scene starts
	StartFacing Heading
	End         Spot // and when it ends
	EndFacing   Heading
	Day         int // the only day of the story it plays on, or 0 for any
	Flags       Flags
}

// Scenes are all the scenes of the story. BUILDING.ADS tags 8 and 9 are
// left out, as in jc_reborn: they are the same as tags 7 and 5. Start and
// end spots of scenes that don't need them are A facing south.
var Scenes = []Scene{
	{"ACTIVITY.ADS", 1, SpotE, SouthEast, SpotA, South, 0, OnIsland | Final | AnyPlace},
	{"ACTIVITY.ADS", 12, SpotD, SouthWest, SpotA, South, 0, OnIsland | Final | AnyPlace | LowTideOK},
	{"ACTIVITY.ADS", 11, SpotA, South, SpotA, South, 0, OnIsland | Final | First | AnyPlace},
	{"ACTIVITY.ADS", 10, SpotD, SouthWest, SpotA, South, 0, OnIsland | Final | AnyPlace | LowTideOK},
	{"ACTIVITY.ADS", 4, SpotE, SouthEast, SpotE, SouthEast, 0, OnIsland | AnyPlace | LowTideOK},
	{"ACTIVITY.ADS", 5, SpotE, SouthWest, SpotA, South, 0, OnIsland | Final | AnyPlace | LowTideOK},
	{"ACTIVITY.ADS", 6, SpotD, SouthWest, SpotA, South, 0, OnIsland | Final | AnyPlace},
	{"ACTIVITY.ADS", 7, SpotD, SouthWest, SpotF, SouthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"ACTIVITY.ADS", 8, SpotA, South, SpotD, SouthEast, 0, OnIsland | First | AnyPlace},
	{"ACTIVITY.ADS", 9, SpotE, East, SpotA, South, 0, OnIsland | Final | LowTideOK},

	{"BUILDING.ADS", 1, SpotF, West, SpotA, West, 0, OnIsland | AnyPlace | LowTideOK},
	{"BUILDING.ADS", 4, SpotA, East, SpotA, South, 0, OnIsland | Final | AnyPlace},
	{"BUILDING.ADS", 3, SpotA, East, SpotC, SouthEast, 0, OnIsland | AnyPlace | LowTideOK},
	{"BUILDING.ADS", 2, SpotF, West, SpotA, South, 0, OnIsland | Final | AnyPlace},
	{"BUILDING.ADS", 5, SpotD, West, SpotD, East, 0, OnIsland | AnyPlace | LowTideOK},
	{"BUILDING.ADS", 7, SpotD, West, SpotD, East, 0, OnIsland | AnyPlace | LowTideOK},
	{"BUILDING.ADS", 6, SpotA, East, SpotA, South, 0, OnIsland | Final | AnyPlace},

	{"FISHING.ADS", 1, SpotD, West, SpotD, East, 0, OnIsland | AnyPlace | LowTideOK},
	{"FISHING.ADS", 2, SpotD, West, SpotD, East, 0, OnIsland | AnyPlace | LowTideOK},
	{"FISHING.ADS", 3, SpotD, West, SpotA, South, 0, OnIsland | Final | AnyPlace | LowTideOK},
	{"FISHING.ADS", 4, SpotE, East, SpotA, South, 0, OnIsland | Final | LeftIsland | LowTideOK},
	{"FISHING.ADS", 5, SpotE, East, SpotA, South, 0, OnIsland | Final | AnyPlace},
	{"FISHING.ADS", 6, SpotD, West, SpotA, South, 0, OnIsland | Final | LowTideOK},
	{"FISHING.ADS", 7, SpotE, East, SpotE, West, 0, OnIsland | LeftIsland | AnyPlace | LowTideOK},
	{"FISHING.ADS", 8, SpotE, East, SpotE, West, 0, OnIsland | LeftIsland | AnyPlace | LowTideOK},

	{"JOHNNY.ADS", 1, SpotA, South, SpotA, South, 11, Final | First},
	{"JOHNNY.ADS", 2, SpotE, SouthWest, SpotF, South, 2, OnIsland | Final | AnyPlace},
	{"JOHNNY.ADS", 3, SpotE, SouthWest, SpotF, NorthEast, 6, OnIsland | AnyPlace | LowTideOK},
	{"JOHNNY.ADS", 4, SpotE, SouthWest, SpotF, NorthEast, 0, OnIsland | AnyPlace},
	{"JOHNNY.ADS", 5, SpotE, SouthWest, SpotF, NorthEast, 0, OnIsland | AnyPlace},
	{"JOHNNY.ADS", 6, SpotA, South, SpotA, South, 10, Final | First},

	{"MARY.ADS", 1, SpotE, SouthWest, SpotA, South, 5, OnIsland | Final | AnyPlace | LowTideOK},
	{"MARY.ADS", 3, SpotF, SouthWest, SpotA, South, 4, OnIsland | Final | First | AnyPlace},
	{"MARY.ADS", 2, SpotE, East, SpotA, South, 1, OnIsland | Final | AnyPlace},
	{"MARY.ADS", 4, SpotE, East, SpotA, South, 7, OnIsland | Final | AnyPlace},
	{"MARY.ADS", 5, SpotE, NorthWest, SpotA, South, 8, OnIsland | LeftIsland | Final | First | NoRaft | AnyPlace},

	{"MISCGAG.ADS", 1, SpotD, West, SpotA, South, 0, OnIsland | Final | AnyPlace | LowTideOK},
	{"MISCGAG.ADS", 2, SpotD, West, SpotA, South, 0, OnIsland | Final | AnyPlace},

	{"STAND.ADS", 1, SpotA, SouthWest, SpotA, SouthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 2, SpotA, West, SpotA, West, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 3, SpotA, NorthWest, SpotA, NorthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 4, SpotB, SouthWest, SpotB, SouthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 5, SpotB, South, SpotB, South, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 6, SpotB, SouthEast, SpotB, SouthEast, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 7, SpotC, NorthEast, SpotC, NorthEast, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 8, SpotC, East, SpotC, East, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 9, SpotD, NorthWest, SpotD, NorthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 10, SpotD, NorthEast, SpotD, NorthEast, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 11, SpotE, NorthWest, SpotE, NorthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 12, SpotF, South, SpotF, South, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 15, SpotA, South, SpotA, South, 0, OnIsland | AnyPlace | LowTideOK},
	{"STAND.ADS", 16, SpotC, South, SpotC, South, 0, OnIsland | AnyPlace | LowTideOK},

	{"SUZY.ADS", 1, SpotA, South, SpotA, South, 3, Final | First},
	{"SUZY.ADS", 2, SpotA, South, SpotA, South, 9, Final | First},

	{"VISITOR.ADS", 1, SpotA, South, SpotA, South, 0, OnIsland | LowTideOK},
	{"VISITOR.ADS", 3, SpotB, NorthWest, SpotD, South, 0, OnIsland | Final | NoHoliday},
	{"VISITOR.ADS", 4, SpotD, South, SpotD, West, 0, OnIsland | AnyPlace | LowTideOK},
	{"VISITOR.ADS", 6, SpotD, South, SpotD, SouthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"VISITOR.ADS", 7, SpotD, South, SpotD, SouthWest, 0, OnIsland | AnyPlace | LowTideOK},
	{"VISITOR.ADS", 5, SpotE, SouthWest, SpotA, South, 0, OnIsland | Final | LeftIsland | AnyPlace | LowTideOK},

	{"WALKSTUF.ADS", 1, SpotA, NorthEast, SpotA, South, 0, OnIsland | Final | LowTideOK},
	{"WALKSTUF.ADS", 2, SpotE, East, SpotD, SouthEast, 0, OnIsland | AnyPlace},
	{"WALKSTUF.ADS", 3, SpotD, West, SpotE, West, 0, OnIsland | AnyPlace | LowTideOK},
}

// Days is how many days the story has. After the last it starts again.
const Days = 11

// Progress is how far the story has come. It is kept between runs of the
// screensaver: the story moves on a day on each new date it runs.
type Progress struct {
	Date string `json:"date"` // the last date it ran, as 2006-01-02
	Day  int    `json:"day"`  // 1 to Days
}

// Advance moves the story on a day if the date is new, and reports whether
// p changed.
func (p *Progress) Advance(now time.Time) bool {
	changed := false
	if d := now.Format(time.DateOnly); d != p.Date {
		p.Date = d
		p.Day++
		changed = true
	}
	if p.Day < 1 || p.Day > Days {
		p.Day = 1
		changed = true
	}
	return changed
}

// leftShift is how far LeftIsland scenes expect the island to the left.
const leftShift = 272

// Holiday is a holiday the island is decorated for.
type Holiday int

const (
	NoHolidayToday Holiday = iota
	Halloween
	StPatricksDay
	Christmas
	NewYear
)

func (h Holiday) String() string {
	return [...]string{"none", "Halloween", "St Patrick's Day", "Christmas", "New Year"}[h]
}

// HolidayOn returns the holiday of date t: Halloween 29-31 October, St
// Patrick's Day 15-17 March, Christmas 23-25 December and New Year 29
// December to 1 January.
func HolidayOn(t time.Time) Holiday {
	md := int(t.Month())*100 + t.Day()
	switch {
	case md >= 1029 && md <= 1031:
		return Halloween
	case md >= 315 && md <= 317:
		return StPatricksDay
	case md >= 1223 && md <= 1225:
		return Christmas
	case md >= 1229 || md <= 101:
		return NewYear
	}
	return NoHolidayToday
}

// NightAt reports whether it is night on the island at time t: two hours in
// every eight, from 23:00 to 1:00, 7:00 to 9:00 and 15:00 to 17:00.
func NightAt(t time.Time) bool {
	h := t.Hour() % 8
	return h == 0 || h == 7
}

// Island is how the island looks for a run of scenes.
type Island struct {
	At      image.Point // offset of the island from its usual place
	LowTide bool
	Raft    int // how much of the raft Johnny has built, 0 to 5
	Night   bool
	Holiday Holiday
}

// RaftFor returns how much of the raft is built on a day of the story.
func RaftFor(day int) int {
	switch {
	case day <= 2:
		return 1
	case day <= 5:
		return day - 1
	}
	return 5
}

// Run is one stretch of the story: the island is set up, some scenes play
// with Johnny walking from one to the next, and a final scene ends it.
type Run struct {
	Island Island
	Scenes []*Scene // the last one is the final scene
}

// Offset is where the scripts of sc draw relative to their own coordinates.
func (r *Run) Offset(sc *Scene) image.Point {
	if sc.Flags&OnIsland == 0 {
		return image.Point{}
	}
	at := r.Island.At
	if sc.Flags&LeftIsland != 0 {
		at.X += leftShift
	}
	return at
}

// pick chooses a scene at random among those with all of want, none of
// unwant, and for any day or for day. It returns nil if there is none.
func pick(rng *rand.Rand, day int, want, unwant Flags) *Scene {
	var ok []*Scene
	for i := range Scenes {
		sc := &Scenes[i]
		if sc.Flags&want == want && sc.Flags&unwant == 0 && (sc.Day == 0 || sc.Day == day) {
			ok = append(ok, sc)
		}
	}
	if len(ok) == 0 {
		return nil
	}
	return ok[rng.IntN(len(ok))]
}

// NewRun plans the next run of the story on day, at time now.
func NewRun(rng *rand.Rand, day int, now time.Time) *Run {
	r := &Run{Island: Island{Night: NightAt(now), Holiday: HolidayOn(now)}}
	final := pick(rng, day, Final, 0)
	if final.Flags&OnIsland != 0 {
		r.Island.place(rng, final, day)
	}

	if final.Flags&First == 0 {
		// The scenes before must suit the island the final scene set up.
		want, unwant := Flags(0), Final
		if r.Island.LowTide {
			want |= LowTideOK
		}
		if r.Island.At != (image.Point{}) {
			want |= AnyPlace
		}
		// jc_reborn draws a new random bound for every scene, so a run has
		// at least 6 scenes but seldom many more than that.
		for i := 0; i < 6+rng.IntN(14); i++ {
			sc := pick(rng, day, want, unwant)
			if sc == nil {
				break
			}
			r.Scenes = append(r.Scenes, sc)
			unwant |= First
		}
	}
	r.Scenes = append(r.Scenes, final)
	return r
}

// place sets up the island for final, the scene that ends the run.
func (is *Island) place(rng *rand.Rand, final *Scene, day int) {
	is.LowTide = final.Flags&LowTideOK != 0 && rng.IntN(2) == 1

	switch {
	case final.Flags&AnyPlace != 0:
		// Three regions of the screen, picked with chances 1/2, 1/4, 1/4.
		switch {
		case rng.IntN(2) == 1:
			is.At = image.Pt(-222+rng.IntN(109), -44+rng.IntN(128))
		case rng.IntN(2) == 1:
			is.At = image.Pt(-114+rng.IntN(134), -14+rng.IntN(99))
		default:
			is.At = image.Pt(-114+rng.IntN(119), -73+rng.IntN(60))
		}
	case final.Flags&LeftIsland != 0:
		is.At = image.Pt(-leftShift, 0)
	}

	if final.Flags&NoRaft == 0 {
		is.Raft = RaftFor(day)
	}
	if final.Flags&NoHoliday != 0 {
		// The cargo ship fills the screen at the end of VISITOR.ADS tag 3,
		// and the decorations would be drawn over it.
		is.Holiday = NoHolidayToday
	}
}
