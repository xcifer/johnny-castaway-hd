package story

import (
	"errors"
	"fmt"
	"image"
	"math/rand/v2"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

func date(month time.Month, day, hour int) time.Time {
	return time.Date(2026, month, day, hour, 30, 0, 0, time.UTC)
}

func TestHolidayOn(t *testing.T) {
	cases := []struct {
		month time.Month
		day   int
		want  Holiday
	}{
		{time.October, 28, NoHolidayToday},
		{time.October, 29, Halloween},
		{time.October, 31, Halloween},
		{time.November, 1, NoHolidayToday},
		{time.March, 14, NoHolidayToday},
		{time.March, 15, StPatricksDay},
		{time.March, 17, StPatricksDay},
		{time.March, 18, NoHolidayToday},
		{time.December, 22, NoHolidayToday},
		{time.December, 23, Christmas},
		{time.December, 25, Christmas},
		{time.December, 26, NoHolidayToday},
		{time.December, 28, NoHolidayToday},
		{time.December, 29, NewYear},
		{time.December, 31, NewYear},
		{time.January, 1, NewYear},
		{time.January, 2, NoHolidayToday},
		{time.July, 4, NoHolidayToday},
	}
	for _, c := range cases {
		if got := HolidayOn(date(c.month, c.day, 12)); got != c.want {
			t.Errorf("%s %d: got %v, want %v", c.month, c.day, got, c.want)
		}
	}

	// Every day of a leap year: 3+3+3+4 holiday days.
	n := 0
	for d := time.Date(2028, 1, 1, 12, 0, 0, 0, time.UTC); d.Year() == 2028; d = d.AddDate(0, 0, 1) {
		if HolidayOn(d) != NoHolidayToday {
			n++
		}
	}
	if n != 13 {
		t.Errorf("%d holiday days in a year, want 13", n)
	}
}

func TestNightAt(t *testing.T) {
	var nights []int
	for h := range 24 {
		if NightAt(date(time.June, 1, h)) {
			nights = append(nights, h)
		}
	}
	if want := []int{0, 7, 8, 15, 16, 23}; !slices.Equal(nights, want) {
		t.Errorf("night at hours %v, want %v", nights, want)
	}
	// The two recordings of the original: 12:23 showed day, 16:11 night.
	if NightAt(time.Date(2026, 10, 28, 12, 23, 53, 0, time.UTC)) {
		t.Error("night at 12:23")
	}
	if !NightAt(time.Date(2026, 10, 28, 16, 11, 14, 0, time.UTC)) {
		t.Error("day at 16:11")
	}
}

func TestRaftFor(t *testing.T) {
	var got []int
	for day := 1; day <= Days; day++ {
		got = append(got, RaftFor(day))
	}
	if want := []int{1, 1, 2, 3, 4, 5, 5, 5, 5, 5, 5}; !slices.Equal(got, want) {
		t.Errorf("raft by day %v, want %v", got, want)
	}
}

func TestScenes(t *testing.T) {
	if len(Scenes) != 63 {
		t.Errorf("%d scenes, want 63", len(Scenes))
	}
	seen := map[string]bool{}
	days := map[int][]string{}
	for _, sc := range Scenes {
		name := fmt.Sprintf("%s %d", sc.ADS, sc.Tag)
		if seen[name] {
			t.Errorf("%s twice", name)
		}
		seen[name] = true
		if sc.Day < 0 || sc.Day > Days {
			t.Errorf("%s: day %d", name, sc.Day)
		}
		if sc.Day > 0 {
			days[sc.Day] = append(days[sc.Day], name)
		}
		// Scenes off the island start and end a run on their own.
		if sc.Flags&OnIsland == 0 && sc.Flags&(Final|First) != Final|First {
			t.Errorf("%s: off the island but flags %08b", name, sc.Flags)
		}
	}
	// Each day of the story has its own scene.
	for day := 1; day <= Days; day++ {
		if len(days[day]) != 1 {
			t.Errorf("day %d: scenes %v, want one", day, days[day])
		}
	}
}

// Whatever the island and the day, there are scenes to fill a run.
func TestPickNeverEmpty(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for day := 1; day <= Days; day++ {
		if pick(rng, day, Final, 0) == nil {
			t.Errorf("day %d: no final scene", day)
		}
		for _, want := range []Flags{0, LowTideOK, AnyPlace, LowTideOK | AnyPlace} {
			for _, unwant := range []Flags{Final, Final | First} {
				if pick(rng, day, want, unwant) == nil {
					t.Errorf("day %d: no scene with %08b without %08b", day, want, unwant)
				}
			}
		}
	}
}

func inRegion(at image.Point) bool {
	regions := []image.Rectangle{
		image.Rect(-222, -44, -222+109, -44+128),
		image.Rect(-114, -14, -114+134, -14+99),
		image.Rect(-114, -73, -114+119, -73+60),
	}
	for _, r := range regions {
		if at.In(r) {
			return true
		}
	}
	return false
}

func TestNewRun(t *testing.T) {
	finals := map[*Scene]int{}
	lengths := map[int]int{}
	lowTide, moved := 0, 0
	const runs = 20000
	for i := range runs {
		rng := rand.New(rand.NewPCG(uint64(i), 7))
		day := 1 + i%Days
		now := date(time.October, 30, 15) // Halloween, night
		r := NewRun(rng, day, now)
		final := r.Scenes[len(r.Scenes)-1]
		finals[final]++
		is := r.Island
		name := fmt.Sprintf("run %d (final %s %d)", i, final.ADS, final.Tag)

		if final.Flags&Final == 0 {
			t.Fatalf("%s: last scene isn't final", name)
		}
		if final.Day != 0 && final.Day != day {
			t.Fatalf("%s: plays on day %d, not %d", name, final.Day, day)
		}
		if !is.Night {
			t.Fatalf("%s: not night", name)
		}

		before := r.Scenes[:len(r.Scenes)-1]
		if final.Flags&First != 0 {
			if len(before) != 0 {
				t.Fatalf("%s: %d scenes before a first scene", name, len(before))
			}
		} else {
			lengths[len(before)]++
		}
		for j, sc := range before {
			if sc.Flags&Final != 0 {
				t.Fatalf("%s: final scene %s %d at %d", name, sc.ADS, sc.Tag, j)
			}
			if j > 0 && sc.Flags&First != 0 {
				t.Fatalf("%s: first scene %s %d at %d", name, sc.ADS, sc.Tag, j)
			}
			if sc.Day != 0 && sc.Day != day {
				t.Fatalf("%s: %s %d on day %d", name, sc.ADS, sc.Tag, day)
			}
			if is.LowTide && sc.Flags&LowTideOK == 0 {
				t.Fatalf("%s: %s %d at low tide", name, sc.ADS, sc.Tag)
			}
			if is.At != (image.Point{}) && sc.Flags&AnyPlace == 0 {
				t.Fatalf("%s: %s %d with the island at %v", name, sc.ADS, sc.Tag, is.At)
			}
		}

		if final.Flags&OnIsland == 0 {
			if is != (Island{Night: true, Holiday: Halloween}) {
				t.Fatalf("%s: off the island, but %+v", name, is)
			}
			continue
		}
		if is.LowTide {
			lowTide++
			if final.Flags&LowTideOK == 0 {
				t.Fatalf("%s: low tide", name)
			}
		}
		switch {
		case final.Flags&AnyPlace != 0:
			moved++
			if !inRegion(is.At) {
				t.Fatalf("%s: island at %v", name, is.At)
			}
		case final.Flags&LeftIsland != 0:
			if is.At != image.Pt(-272, 0) {
				t.Fatalf("%s: island at %v", name, is.At)
			}
		default:
			if is.At != (image.Point{}) {
				t.Fatalf("%s: island at %v", name, is.At)
			}
		}
		wantRaft := RaftFor(day)
		if final.Flags&NoRaft != 0 {
			wantRaft = 0
		}
		if is.Raft != wantRaft {
			t.Fatalf("%s: raft %d, want %d", name, is.Raft, wantRaft)
		}
		wantHoliday := Halloween
		if final.Flags&NoHoliday != 0 {
			wantHoliday = NoHolidayToday
		}
		if is.Holiday != wantHoliday {
			t.Fatalf("%s: holiday %v", name, is.Holiday)
		}
	}

	// Every final scene comes up.
	for i := range Scenes {
		if sc := &Scenes[i]; sc.Flags&Final != 0 && finals[sc] == 0 {
			t.Errorf("final scene %s %d never picked", sc.ADS, sc.Tag)
		}
	}
	// 6 to 19 scenes before the final one. The loop goes on past scene k
	// with chance (19-k)/14, so runs have 8 to 10 scenes most often.
	total := 0
	for n, c := range lengths {
		if n < 6 || n > 19 {
			t.Errorf("%d runs with %d scenes before the final one", c, n)
		}
		total += c
	}
	p := 1.0
	for n := 6; n <= 19; n++ {
		want := p * float64(n-5) / 14
		p -= want
		if got := float64(lengths[n]) / float64(total); got < want-0.015 || got > want+0.015 {
			t.Errorf("%.3f of runs have %d scenes before the final one, want %.3f", got, n, want)
		}
	}
	if lowTide == 0 || moved == 0 {
		t.Errorf("low tide %d, moved %d times", lowTide, moved)
	}
}

func TestRunOffset(t *testing.T) {
	r := &Run{Island: Island{At: image.Pt(-30, 12)}}
	cases := []struct {
		flags Flags
		want  image.Point
	}{
		{OnIsland, image.Pt(-30, 12)},
		{OnIsland | LeftIsland, image.Pt(242, 12)},
		{Final | First, image.Pt(0, 0)},
	}
	for _, c := range cases {
		if got := r.Offset(&Scene{Flags: c.flags}); got != c.want {
			t.Errorf("flags %08b: offset %v, want %v", c.flags, got, c.want)
		}
	}
}

// Every scene of the story is in the game data, and the scenes the story
// leaves out are the ones it should.
func TestGameDataScenes(t *testing.T) {
	a, err := resource.Load("../../gamedata")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	if err != nil {
		t.Fatal(err)
	}
	inStory := map[string]bool{}
	for _, sc := range Scenes {
		inStory[fmt.Sprintf("%s %d", sc.ADS, sc.Tag)] = true
	}
	var missing []string
	for i := range a.Entries {
		e := &a.Entries[i]
		if e.Type() != "ADS" {
			continue
		}
		ads, err := resource.DecodeADS(e)
		if err != nil {
			t.Fatal(err)
		}
		for _, tag := range ads.Tags {
			name := fmt.Sprintf("%s %d", e.Name, tag.ID)
			if inStory[name] {
				delete(inStory, name)
			} else {
				missing = append(missing, name)
			}
		}
	}
	for name := range inStory {
		t.Errorf("%s is in the story but not in the data", name)
	}
	slices.Sort(missing)
	if want := []string{"BUILDING.ADS 8", "BUILDING.ADS 9", "STAND.ADS 14"}; !slices.Equal(missing, want) {
		t.Errorf("scenes not in the story: %v, want %v", missing, want)
	}
}
