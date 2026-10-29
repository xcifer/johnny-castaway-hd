package script

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// label is a tag label in ADS code.
func label(id uint16) resource.Instr { return resource.Instr{Opcode: id} }

// clips is the TTM for scene tests. Clips 1-5 show one frame (sample n)
// and end one redraw later; clip 9 shows its frame for 10 redraws.
func clips() *Script {
	var code []resource.Instr
	for n := uint16(1); n <= 5; n++ {
		code = append(code, ins(opTag, n), ins(opPlaySample, n), ins(opUpdate), ins(opPurge), ins(opUpdate))
	}
	code = append(code,
		ins(opTag, 9), ins(opSetDelay, 40), ins(opPlaySample, 9), ins(opUpdate), ins(opPurge), ins(opUpdate),
		ins(opTag, 99), ins(opUpdate))
	return newTestScript(code...)
}

// playScene plays scene 1 of code with clips() in slot 1, and returns the
// samples played (clip starts, "n@redraw") and the redraw it ended on.
func playScene(t *testing.T, seed uint64, code ...resource.Instr) (*harness, int) {
	t.Helper()
	h := newHarness(t)
	h.s.Rand = rand.New(rand.NewPCG(seed, 0))
	sc, err := NewScene(h.s, "TEST.ADS", &resource.ADS{Code: code}, map[uint16]*Script{1: clips()}, 1)
	if err != nil {
		t.Fatal(err)
	}
	for ; h.redraw < 500; h.redraw++ {
		if h.redraw > 0 {
			sc.Step()
		}
		if sc.Done() {
			return h, h.redraw
		}
		h.s.Play()
	}
	t.Fatal("scene still playing after 500 redraws")
	return nil, 0
}

func add(slot, tag uint16, n int16) resource.Instr {
	return ins(adsAddScene, slot, tag, uint16(n), 1)
}

// A scene starts with its first block, and each clip that ends runs the
// blocks waiting for it, in the same redraw. Blocks under other tags
// belong to other scenes.
func TestSceneSequence(t *testing.T) {
	h, end := playScene(t, 0,
		label(1),
		add(1, 1, 0),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 1),
		add(1, 2, 0),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 2),
		add(1, 3, 0),
		ins(adsPlayScene),
		label(2),
		ins(adsIfLastPlayed, 1, 3),
		add(1, 4, 0),
		ins(adsPlayScene),
	)
	h.want("samples", h.samples, "1@0", "2@2", "3@4")
	if end != 6 {
		t.Errorf("scene ended at redraw %d, want 6", end)
	}
}

// IF_LASTPLAYED a OR IF_LASTPLAYED b: the block runs after either clip.
func TestSceneOr(t *testing.T) {
	code := func(first uint16) []resource.Instr {
		return []resource.Instr{
			label(1),
			add(1, first, 0),
			ins(adsPlayScene),
			ins(adsIfLastPlayed, 1, 1),
			ins(adsOr),
			ins(adsIfLastPlayed, 1, 2),
			add(1, 3, 0),
			ins(adsIfLastPlayed, 1, 4), // no OR before it: ends the block
			add(1, 5, 0),
			ins(adsPlayScene),
		}
	}
	h, _ := playScene(t, 0, code(1)...)
	h.want("after 1", h.samples, "1@0", "3@2")
	h, _ = playScene(t, 0, code(2)...)
	h.want("after 2", h.samples, "2@0", "3@2")
}

// A clip played several times triggers its blocks once, after the last.
func TestSceneRepeats(t *testing.T) {
	h, _ := playScene(t, 0,
		label(1),
		add(1, 1, 3),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 1),
		add(1, 2, 0),
		ins(adsPlayScene),
	)
	h.want("samples", h.samples, "1@0", "1@2", "1@4", "2@6")
}

// RANDOM_START ... RANDOM_END carries out one option, picked by weight.
func TestSceneRandom(t *testing.T) {
	count := map[string]int{}
	for seed := range uint64(600) {
		h, _ := playScene(t, seed,
			label(1),
			ins(adsRandomStart),
			ins(adsAddScene, 1, 1, 0, 3),
			ins(adsAddScene, 1, 2, 0, 1),
			ins(adsAddScene, 1, 3, 0, 0),
			ins(adsNop, 2),
			ins(adsRandomEnd),
			ins(adsPlayScene),
		)
		count[fmt.Sprint(h.samples)]++
	}
	// Expected 300, 100, 0 and 200 of 600.
	if c := count["[1@0]"]; c < 250 || c > 350 {
		t.Errorf("clip 1 (weight 3) chosen %d times of 600", c)
	}
	if c := count["[2@0]"]; c < 65 || c > 135 {
		t.Errorf("clip 2 (weight 1) chosen %d times of 600", c)
	}
	if c := count["[3@0]"]; c != 0 {
		t.Errorf("clip 3 (weight 0) chosen %d times", c)
	}
	if c := count["[]"]; c < 150 || c > 250 {
		t.Errorf("NOP (weight 2) chosen %d times of 600", c)
	}
}

// IF_NOT_RUNNING and IF_IS_RUNNING skip up to the next PLAY_SCENE or END,
// which then only ends the skip.
func TestSceneConditions(t *testing.T) {
	h, _ := playScene(t, 0,
		label(1),
		add(1, 9, 0),
		ins(adsIfNotRunning, 1, 9),
		add(1, 1, 0), // skipped: 9 is running
		ins(adsPlayScene),
		ins(adsIfIsRunning, 1, 4),
		add(1, 3, 0), // skipped: 4 isn't running
		ins(adsEnd),  // ends the skip, not the scene
		ins(adsIfIsRunning, 1, 9),
		add(1, 2, 0),
		ins(adsPlayScene), // ends the block
		add(1, 5, 0),
		ins(adsIfLastPlayed, 1, 2),
		add(1, 4, 0),
		ins(adsPlayScene),
	)
	// The IF_NOT_RUNNING 1 9 at the start also waits for clip 9: when it
	// ends, its block starts clip 1.
	h.want("samples", h.samples, "9@0", "2@0", "4@2", "1@20")
}

// A clip showing its last frame counts as not running.
func TestSceneEndingNotRunning(t *testing.T) {
	h, _ := playScene(t, 0,
		label(1),
		add(1, 9, 0), // last frame shown from redraw 10 to 20
		add(1, 1, 6), // ends at redraw 12
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 1),
		ins(adsIfNotRunning, 1, 9),
		add(1, 2, 0),
		ins(adsPlayScene),
	)
	h.want("samples", h.samples, "9@0", "1@0", "1@2", "1@4", "1@6", "1@8", "1@10", "2@12")
}

// STOP_SCENE stops a clip at once, without running its blocks.
func TestSceneStop(t *testing.T) {
	h, end := playScene(t, 0,
		label(1),
		add(1, 9, 0),
		add(1, 1, 0),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 1),
		ins(adsStopScene, 1, 9, 1),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 9),
		add(1, 3, 0),
		ins(adsPlayScene),
	)
	h.want("samples", h.samples, "9@0", "1@0")
	if end != 2 {
		t.Errorf("scene ended at redraw %d, want 2", end)
	}
}

// After END, clips that end start nothing more.
func TestSceneEnd(t *testing.T) {
	h, _ := playScene(t, 0,
		label(1),
		add(1, 1, 0),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 1),
		add(1, 2, 0),
		ins(adsFadeOut),
		ins(adsEnd),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 2),
		add(1, 3, 0),
		ins(adsPlayScene),
	)
	h.want("samples", h.samples, "1@0", "2@2")
}

// GOSUB_TAG runs the block of another tag.
func TestSceneGosub(t *testing.T) {
	h, _ := playScene(t, 0,
		label(1),
		ins(adsGosubTag, 14),
		add(1, 1, 0),
		ins(adsPlayScene),
		label(14),
		add(1, 4, 0),
		ins(adsPlayScene),
	)
	h.want("samples", h.samples, "4@0", "1@0")
}

// The IF_NOT_RUNNINGs at the start of a scene also wait for their clip:
// the idle scenes in STAND.ADS pick a new clip whenever one ends.
func TestSceneLeadingIfNotRunning(t *testing.T) {
	h := newHarness(t)
	h.s.Rand = rand.New(rand.NewPCG(1, 0))
	code := []resource.Instr{
		label(1),
		ins(adsIfNotRunning, 1, 1),
		ins(adsAnd),
		ins(adsIfNotRunning, 1, 2),
		ins(adsRandomStart),
		ins(adsAddScene, 1, 1, 0, 1),
		ins(adsAddScene, 1, 2, 0, 1),
		ins(adsRandomEnd),
		ins(adsPlayScene),
		ins(adsIfIsRunning, 1, 9),
		ins(adsIfNotRunning, 1, 3), // not waiting: it comes after IF_IS_RUNNING
		add(1, 5, 0),
		ins(adsPlayScene),
	}
	sc, err := NewScene(h.s, "TEST.ADS", &resource.ADS{Code: code}, map[uint16]*Script{1: clips()}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.triggers) != 2 {
		t.Errorf("%d triggers, want 2 (clips 1 and 2)", len(sc.triggers))
	}
	// Each clip that ends starts another, forever.
	for ; h.redraw < 40; h.redraw++ {
		if h.redraw > 0 {
			sc.Step()
		}
		h.s.Play()
	}
	if len(h.samples) != 20 {
		t.Errorf("%d clips started in 40 redraws, want 20: %v", len(h.samples), h.samples)
	}
	for i, smp := range h.samples {
		if smp != fmt.Sprintf("1@%d", 2*i) && smp != fmt.Sprintf("2@%d", 2*i) {
			t.Errorf("clip start %d: %s", i, smp)
		}
	}
}

// IF_LASTPLAYED_LOCAL (ACTIVITY.ADS tag 7) waits for a clip in place of the
// scene's own blocks, once; its ADD_SCENE_LOCAL starts the next clip.
func TestSceneLocal(t *testing.T) {
	h, end := playScene(t, 0,
		label(1),
		add(1, 1, 0),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 1),
		add(1, 2, 0),
		ins(adsIfLastPlayedLocal, 1, 2),
		ins(adsAddSceneLocal, adsAddScene, 1, 3, 0, 1),
		ins(adsPlayScene),
		ins(adsIfLastPlayed, 1, 2), // replaced by the local block
		add(1, 4, 0),
		ins(adsPlayScene),
	)
	h.want("samples", h.samples, "1@0", "2@2", "3@4")
	if end != 6 {
		t.Errorf("scene ended at redraw %d, want 6", end)
	}
}

func TestSceneMissing(t *testing.T) {
	h := newHarness(t)
	if _, err := NewScene(h.s, "TEST.ADS", &resource.ADS{Code: []resource.Instr{label(1), ins(adsPlayScene)}}, nil, 2); err == nil {
		t.Error("no error for a missing scene")
	}
}
