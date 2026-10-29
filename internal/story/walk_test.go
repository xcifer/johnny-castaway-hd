package story

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

var update = flag.Bool("update", false, "rewrite testdata/walks.md5 from the game data")

func TestRoutes(t *testing.T) {
	for from := range Spot(numSpots) {
		for to := range Spot(numSpots) {
			routes := Routes(from, to)
			if len(routes) == 0 || len(routes) > 50 {
				t.Errorf("%v to %v: %d routes", from, to, len(routes))
			}
			if from == to {
				if len(routes) != 1 || len(routes[0]) != 1 {
					t.Errorf("%v to itself: %v", from, routes)
				}
				continue
			}
			for _, r := range routes {
				if r[0] != from || r[len(r)-1] != to {
					t.Errorf("%v to %v: route %v", from, to, r)
				}
				seen := map[Spot]bool{}
				for i, s := range r {
					if seen[s] {
						t.Errorf("%v to %v: route %v passes %v twice", from, to, r, s)
					}
					seen[s] = true
					if i == 0 {
						continue
					}
					a := r[i-1]
					if walkStart[a][s] < 0 || setOffFacing[a][s] < 0 || arriveFacing[a][s] < 0 {
						t.Errorf("%v to %v: route %v has no walk from %v to %v", from, to, r, a, s)
					}
				}
			}
		}
	}
	// Worked out by hand: B is left for A or C; from C on to D; from A only
	// on to E; from E to D or F; from F to D.
	want := [][]Spot{{SpotB, SpotA, SpotE, SpotD}, {SpotB, SpotA, SpotE, SpotF, SpotD}, {SpotB, SpotC, SpotD}}
	if got := Routes(SpotB, SpotD); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("B to D: %v", got)
	}
}

func loadWalkTable(t *testing.T) []resource.WalkStep {
	t.Helper()
	exe, err := os.ReadFile("../../gamedata/SCRANTIC.SCR")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	if err != nil {
		t.Fatal(err)
	}
	table, err := resource.WalkTable(exe)
	if err != nil {
		t.Fatal(err)
	}
	return table
}

// The table's layout: each walk and each set of turns ends with a step
// whose X is 0, and the turns come in two sets of 8 headings.
func TestGameDataWalkTable(t *testing.T) {
	table := loadWalkTable(t)
	end := func(i int) bool { return i < 0 || table[i].X == 0 }
	for from := range numSpots {
		for to := range numSpots {
			if i := walkStart[from][to]; i >= 0 && (!end(i-1) || end(i)) {
				t.Errorf("walk %v to %v at %d doesn't start after an end", Spot(from), Spot(to), i)
			}
		}
		i := turnStart[from]
		for k := range 2*pocketsOffset - 1 {
			if end(i+k) != (k == 8) {
				t.Errorf("turns of %v: step %d", Spot(from), i+k)
			}
		}
		if !end(i-1) || !end(i+2*pocketsOffset-1) {
			t.Errorf("turns of %v at %d aren't between ends", Spot(from), i)
		}
	}
}

// Every walk, along every route and between all headings, ends turned the
// right way at the right spot, and Johnny never jumps.
func TestGameDataWalks(t *testing.T) {
	table := loadWalkTable(t)
	// The table repeats one step itself (C to A, rows 188 and 189).
	repeated := map[resource.WalkStep]bool{}
	for i := 1; i < len(table); i++ {
		if table[i] == table[i-1] && table[i].X != 0 {
			repeated[table[i]] = true
		}
	}
	if len(repeated) != 1 {
		t.Errorf("%d steps repeated in the table, want 1", len(repeated))
	}
	walks, frames := 0, 0
	maxJump := 0
	for from := range Spot(numSpots) {
		for to := range Spot(numSpots) {
			for _, route := range Routes(from, to) {
				for fh := range Heading(8) {
					for th := range Heading(8) {
						w := walkRoute(table, route, fh, th)
						walks++
						frames += len(w)
						last := w[len(w)-1]
						if want := table[turnStart[to]+int(th)+pocketsOffset]; last.WalkStep != want {
							t.Fatalf("%v %v->%v %v: ends with %+v, want %+v", route, fh, to, th, last.WalkStep, want)
						}
						behind := false
						for i, f := range w {
							if f.X == 0 {
								t.Fatalf("%v %v->%v: step %d is an end marker", route, fh, th, i)
							}
							behind = behind || f.BehindTree
							if i > 0 {
								p := w[i-1]
								if f == p && !repeated[f.WalkStep] {
									t.Fatalf("%v %v->%v: step %d shown twice", route, fh, th, i)
								}
								maxJump = max(maxJump, abs(f.X-p.X), abs(f.Y-p.Y))
							}
						}
						if behind != hasLeg(route, SpotD, SpotE) {
							t.Fatalf("%v: behind the tree %v", route, behind)
						}
						if from == to {
							// Turning on the spot the shorter way, stopping a step
							// early, then the picture standing still.
							d := int(th-fh) & 7
							if want := max(0, min(d, 8-d)-1) + 1; len(w) != want {
								t.Fatalf("%v turning %v to %v: %d pictures, want %d", from, fh, th, len(w), want)
							}
						}
					}
				}
			}
		}
	}
	// Positions are top-left corners. Strides in the table itself reach 17
	// px; arriving at E and turning moves 20 (16 at the sprites' centers).
	// A wrong step in the table would land at another spot, ~100 px away.
	t.Logf("%d walks, %d pictures, largest move %d px", walks, frames, maxJump)
	if maxJump > 20 {
		t.Errorf("Johnny jumps %d px between two pictures", maxJump)
	}
}

func hasLeg(route []Spot, a, b Spot) bool {
	for i := 1; i < len(route); i++ {
		if route[i-1] == a && route[i] == b || route[i-1] == b && route[i] == a {
			return true
		}
	}
	return false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestWalkPicksRoutes(t *testing.T) {
	table := loadWalkTable(t)
	rng := rand.New(rand.NewPCG(3, 4))
	lengths := map[int]bool{}
	for range 200 {
		lengths[len(Walk(table, rng, SpotA, South, SpotD, West))] = true
	}
	if n := len(Routes(SpotA, SpotD)); n > 1 && len(lengths) < 2 {
		t.Errorf("%d routes from A to D, but every walk was the same length", n)
	}
}

// Every walk is drawn as before: one hash per pair of spots, over all its
// routes and headings (testdata/walks.md5; run with -update to rewrite it).
func TestGameDataWalksGolden(t *testing.T) {
	table := loadWalkTable(t)
	var lines []string
	for from := range Spot(numSpots) {
		for to := range Spot(numSpots) {
			h := md5.New()
			routes := Routes(from, to)
			n := 0
			for _, route := range routes {
				for fh := range Heading(8) {
					for th := range Heading(8) {
						w := walkRoute(table, route, fh, th)
						n += len(w)
						fmt.Fprintf(h, "%v %d %d:", route, fh, th)
						for _, f := range w {
							fmt.Fprintf(h, " %v", f)
						}
						fmt.Fprintln(h)
					}
				}
			}
			lines = append(lines, fmt.Sprintf("%v %v routes %d pictures %d %x", from, to, len(routes), n, h.Sum(nil)))
		}
	}
	got := strings.Join(lines, "\n") + "\n"
	golden := filepath.Join("testdata", "walks.md5")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("walks differ from testdata/walks.md5:\n%s", got)
	}
}
