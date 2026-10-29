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
	expected := regexp.MustCompile(`isn't loaded$|^FIRE\.TTM at \d+: (FLAME|FLURRY)\.BMP not found$|^SASKDATE\.TTM at 4638: color 207 outside the palette, using 15$`)
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
	got := strings.Join(lines, "\n") + "\n"

	golden := filepath.Join("testdata", "scripts.md5")
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
		t.Error("output differs from testdata/scripts.md5")
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
