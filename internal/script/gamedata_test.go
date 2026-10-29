package script

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

var update = flag.Bool("update", false, "rewrite testdata/scripts.md5 from the game data")

func loadGameData(t *testing.T) (*resource.Archive, color.Palette) {
	t.Helper()
	a, err := resource.Load("../../gamedata")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	if err != nil {
		t.Fatal(err)
	}
	p, err := resource.DecodePAL(a.Find("JOHNCAST.PAL"))
	if err != nil {
		t.Fatal(err)
	}
	var pal color.Palette
	for i := range 16 {
		pal = append(pal, p.Color(i))
	}
	return a, pal
}

func loadScripts(t *testing.T, a *resource.Archive) map[string]*Script {
	t.Helper()
	scripts := map[string]*Script{}
	for i := range a.Entries {
		e := &a.Entries[i]
		if e.Type() != "TTM" {
			continue
		}
		ttm, err := resource.DecodeTTM(e)
		if err != nil {
			t.Fatal(err)
		}
		scripts[e.Name] = NewScript(e.Name, ttm)
	}
	return scripts
}

// Played start to finish, every script must end, use no unknown opcode
// and draw exactly as before (testdata/scripts.md5; run with -update to
// rewrite it). Playing a whole script runs its clips in file order rather
// than the order scenes use, so a clip can come before the one that loads
// its pictures: those warnings are expected here.
func TestGameDataScripts(t *testing.T) {
	a, pal := loadGameData(t)
	expected := regexp.MustCompile(`isn't loaded$|\.BMP has no sprite \d+$|^FIRE\.TTM at \d+: (FLAME|FLURRY)\.BMP not found$|^SASKDATE\.TTM at 4638: color 207 outside the palette, using 15$`)
	scripts := loadScripts(t, a)
	if len(scripts) != 41 {
		t.Fatalf("%d scripts, want 41", len(scripts))
	}

	var lines []string
	for i := range a.Entries {
		name := a.Entries[i].Name
		sc := scripts[name]
		if sc == nil {
			continue
		}
		p := NewPaletted(a, pal)
		s := NewScheduler(p)
		s.Logf = func(format string, args ...any) {
			if msg := fmt.Sprintf(format, args...); !expected.MatchString(msg) {
				t.Errorf("unexpected: %s", msg)
			}
		}
		samples := 0
		s.Sound = func(int) { samples++ }
		s.Rand = rand.New(rand.NewPCG(1, 2))
		s.AddScript(sc)

		// Fingerprint every tenth frame and the last one.
		h := md5.New()
		snapshot := func() {
			h.Write(p.background)
			h.Write(p.saved.pix)
			for _, l := range p.layers {
				if !l.hidden {
					h.Write(l.pix)
				}
			}
		}
		redraws, frames := 0, 0
		for ; len(s.Threads()) > 0; redraws++ {
			if redraws == 100000 {
				t.Fatalf("%s still playing after %d redraws", name, redraws)
			}
			s.Step()
			if th := s.Threads(); len(th) == 1 && !th[0].shown {
				if frames%10 == 0 {
					snapshot()
				}
				frames++
			}
			s.Play()
		}
		snapshot()
		lines = append(lines, fmt.Sprintf("%s ticks %d frames %d samples %d %x", name, redraws*FrameTicks, frames, samples, h.Sum(nil)))
	}
	compareGolden(t, "scripts.md5", lines)
}

// compareGolden compares lines with testdata/name, or rewrites it with
// -update.
func compareGolden(t *testing.T, name string, lines []string) {
	t.Helper()
	got := strings.Join(lines, "\n") + "\n"
	golden := filepath.Join("testdata", name)
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
		wantLines := map[string]bool{}
		for _, l := range strings.Split(string(want), "\n") {
			wantLines[strings.TrimSpace(l)] = true
		}
		for _, l := range lines {
			if !wantLines[l] {
				t.Errorf("changed or new: %s", l)
			}
		}
		t.Errorf("output differs from testdata/%s", name)
	}
}

// Every clip a scene script starts or stops must exist, and nothing the
// interpreter treats as impossible may occur in the data.
func TestGameDataClips(t *testing.T) {
	a, _ := loadGameData(t)
	scripts := loadScripts(t, a)
	refs := 0
	for i := range a.Entries {
		e := &a.Entries[i]
		if e.Type() != "ADS" {
			continue
		}
		ads, err := resource.DecodeADS(e)
		if err != nil {
			t.Fatal(err)
		}
		slots := map[uint16]string{}
		for _, r := range ads.Resources {
			slots[r.ID] = r.Name
		}
		for _, in := range ads.Code {
			if in.Opcode != 0x2005 && in.Opcode != 0x2010 && in.Opcode != 0x1520 {
				continue
			}
			slot, tag := in.Args[0], in.Args[1]
			if in.Opcode == 0x1520 {
				// Its five words are 0x2005 and the arguments of an
				// ordinary ADD_SCENE.
				slot, tag = in.Args[1], in.Args[2]
			}
			sc := scripts[slots[slot]]
			if sc == nil {
				t.Errorf("%s at %d: no script in slot %d", e.Name, in.Offset, slot)
				continue
			}
			if _, ok := sc.findTag(tag); !ok {
				t.Errorf("%s at %d: %s has no tag %d", e.Name, in.Offset, sc.Name, tag)
			}
			refs++
		}
	}
	if refs < 900 {
		t.Errorf("only %d clip references checked", refs)
	}

	for _, sc := range scripts {
		for _, in := range sc.code {
			switch in.Opcode {
			case opTimer:
				if in.Args[0] > in.Args[1] {
					t.Errorf("%s at %d: TIMER %v has its range backwards", sc.Name, in.Offset, in.Args)
				}
			case opDrawCircle:
				if in.Args[2] != in.Args[3] || in.Args[2]%2 != 0 || in.Args[2] < 2 {
					t.Errorf("%s at %d: circle %dx%d", sc.Name, in.Offset, in.Args[2], in.Args[3])
				}
			case opSetBmpSlot:
				if in.Args[0] >= bmpSlots {
					t.Errorf("%s at %d: bitmap slot %d", sc.Name, in.Offset, in.Args[0])
				}
			}
		}
	}
}

// Warnings the game data causes in its scenes. Each is a mistake in the
// original scripts; see docs/ads-opcodes.md.
var sceneWarnings = map[string]bool{
	"MJDIVE.TTM at 66: sprite 9 of bitmap slot 1 isn't loaded":      true,
	"GJVIS5.TTM at 3478: sprite 0 of bitmap slot 1 isn't loaded":    true,
	"GJGULIVR.TTM at 3614: sprite 26 of bitmap slot 1 isn't loaded": true,
	"GJNAT1.TTM at 518: GJNAT1LI.BMP has no sprite 19":              true,
	"GJNAT1.TTM at 6556: GJNAT1.BMP has no sprite 27":               true,
	"GJVIS3.TTM at 4356: GJVIS3.BMP has no sprite 12":               true,
	"GJVIS3.TTM at 4496: GJVIS3.BMP has no sprite 12":               true,
	"SMDATE.TTM at 6332: SMDATE11.BMP has no sprite 31":             true,
	"SMDATE.TTM at 6362: SMDATE11.BMP has no sprite 30":             true,
	"SMDATE.TTM at 6388: SMDATE11.BMP has no sprite 31":             true,
	"SASKDATE.TTM at 1928: SJGFTASK.BMP has no sprite 41":           true,
	"SASKDATE.TTM at 4638: color 207 outside the palette, using 15": true,
}

// Every scene of every scene script must end, with no warnings but the
// known ones, and start its clips as before (testdata/scenes.md5, which
// fingerprints the clips started and when, for five random seeds; run
// with -update to rewrite it).
func TestGameDataScenes(t *testing.T) {
	a, pal := loadGameData(t)
	var lines []string
	seen := map[string]bool{}
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
			h := md5.New()
			longest := 0
			for seed := range uint64(5) {
				scripts := map[uint16]*Script{}
				for _, r := range ads.Resources {
					ttm, err := resource.DecodeTTM(a.Find(r.Name))
					if err != nil {
						t.Fatal(err)
					}
					scripts[r.ID] = NewScript(r.Name, ttm)
				}
				s := NewScheduler(NewPaletted(a, pal))
				s.Rand = rand.New(rand.NewPCG(seed, 0))
				s.Logf = func(format string, args ...any) {
					msg := fmt.Sprintf(format, args...)
					seen[msg] = true
					if !sceneWarnings[msg] {
						t.Errorf("%s scene %d: %s", e.Name, tag.ID, msg)
					}
				}
				sc, err := NewScene(s, e.Name, ads, scripts, tag.ID)
				if err != nil {
					t.Fatal(err)
				}
				var playing []*Thread
				redraw := 0
				for ; !sc.Done(); redraw++ {
					if redraw == 20000 {
						t.Fatalf("%s scene %d (seed %d) still playing after %d redraws", e.Name, tag.ID, seed, redraw)
					}
					if redraw > 0 {
						sc.Step()
					}
					s.Play()
					for _, th := range s.Threads() {
						if !slices.Contains(playing, th) {
							fmt.Fprintf(h, "%s:%d@%d ", th.Script.Name, th.Tag, redraw)
						}
					}
					playing = s.Threads()
				}
				fmt.Fprintf(h, "end@%d\n", redraw)
				longest = max(longest, redraw)
			}
			lines = append(lines, fmt.Sprintf("%s %d longest %d %x", e.Name, tag.ID, longest, h.Sum(nil)))
		}
	}
	if len(lines) != 66 {
		t.Errorf("%d scenes, want 66", len(lines))
	}
	for msg := range sceneWarnings {
		if !seen[msg] {
			t.Errorf("expected warning never seen: %s", msg)
		}
	}
	compareGolden(t, "scenes.md5", lines)
}
