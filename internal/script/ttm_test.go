package script

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"testing"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

func ins(op uint16, args ...uint16) resource.Instr {
	return resource.Instr{Opcode: op, Args: args}
}

func str(op uint16, s string) resource.Instr {
	return resource.Instr{Opcode: op, Str: s, HasStr: true}
}

func newTestScript(code ...resource.Instr) *Script {
	for i := range code {
		code[i].Offset = i
	}
	return NewScript("TEST.TTM", &resource.TTM{Code: code})
}

// testPalette maps index i to gray level 10+i, so no color is black.
func testPalette() color.Palette {
	pal := make(color.Palette, 16)
	for i := range pal {
		g := uint8(10 + i)
		pal[i] = color.NRGBA{g, g, g, 0xFF}
	}
	return pal
}

// harness runs a scheduler on a Paletted canvas and records, per tick,
// the samples played (PLAY_SAMPLE marks frames in test scripts) and the
// threads that stopped.
type harness struct {
	t       *testing.T
	s       *Scheduler
	p       *Paletted
	tick    int
	samples []string
	stopped []string
	logs    []string
}

func newHarness(t *testing.T) *harness {
	h := &harness{t: t, p: NewPaletted(nil, testPalette())}
	h.s = NewScheduler(h.p)
	h.s.Sound = func(n int) { h.samples = append(h.samples, fmt.Sprintf("%d@%d", n, h.tick)) }
	h.s.Logf = func(format string, args ...any) { h.logs = append(h.logs, fmt.Sprintf(format, args...)) }
	return h
}

// run plays ticks, starting with tick 0 for threads already added.
func (h *harness) run(ticks int) {
	for range ticks {
		if h.tick > 0 {
			for _, t := range h.s.Step() {
				h.stopped = append(h.stopped, fmt.Sprintf("%d@%d", t.Tag, h.tick))
			}
		}
		h.s.Play()
		h.tick++
	}
}

func (h *harness) want(what string, got []string, want ...string) {
	h.t.Helper()
	if !slices.Equal(got, want) {
		h.t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

func TestFrameTiming(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(opSetDelay, 6),
		ins(opPlaySample, 1),
		ins(opUpdate),
		ins(opPlaySample, 2),
		ins(opUpdate),
		ins(opPurge),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.s.Add(sc, 1, 0)
	h.run(30)
	// The last frame stays up for its delay after PURGE, then the clip stops.
	h.want("samples", h.samples, "1@0", "2@6")
	h.want("stopped", h.stopped, "1@18")
	if len(h.s.Threads()) != 0 || !h.p.layers[0].hidden {
		t.Error("stopped thread still there or its layer still visible")
	}
}

func TestMinimumDelay(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(opSetDelay, 0),
		ins(opPlaySample, 1),
		ins(opUpdate),
		ins(opTimer, 3, 10), // (3+10)/2 = 6
		ins(opPlaySample, 2),
		ins(opUpdate),
		ins(opPlaySample, 3),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.s.Add(sc, 1, 0)
	h.run(30)
	h.want("samples", h.samples, "1@0", "2@4", "3@10")
	// The code ends right after the last UPDATE: that frame is the last.
	h.want("stopped", h.stopped, "1@16")
}

func TestGotoTag(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(opPlaySample, 1),
		ins(opGotoTag, 2), // taken after this frame, not at once
		ins(opPlaySample, 2),
		ins(opUpdate),
		ins(opPlaySample, 3),
		ins(opUpdate),
		ins(opTag, 2),
		ins(opPlaySample, 4),
		ins(opUpdate),
		ins(opPurge),
		ins(opUpdate),
		ins(opGotoTag, 99),
	)
	h := newHarness(t)
	h.s.Add(sc, 1, 0)
	h.run(20)
	h.want("samples", h.samples, "1@0", "2@0", "4@4")
	h.want("stopped", h.stopped, "1@12")
}

func TestRepeats(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(opPlaySample, 1),
		ins(opUpdate),
		ins(opPurge),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.s.Add(sc, 1, 3)
	h.run(40)
	h.want("samples", h.samples, "1@0", "1@8", "1@16")
	h.want("stopped", h.stopped, "1@24")
}

// A negative count loops the clip for that many ticks: PURGE jumps back to
// the start of the clip instead of ending it.
func TestSceneTimer(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(opPlaySample, 1),
		ins(opUpdate),
		ins(opPurge),
		ins(opUpdate),
		ins(opTag, 2),
		ins(opPlaySample, 2),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.s.Add(sc, 1, -20)
	h.run(40)
	h.want("samples", h.samples, "1@0", "1@8", "1@16")
	h.want("stopped", h.stopped, "1@20")
}

// A timed clip that isn't the first in its script loops back to its own
// start.
func TestSceneTimerLaterClip(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(opPlaySample, 1),
		ins(opUpdate),
		ins(opTag, 2),
		ins(opPlaySample, 2),
		ins(opUpdate),
		ins(opPurge),
		ins(opUpdate),
		ins(opTag, 3), // running off the end would stop the clip
		ins(opUpdate),
	)
	h := newHarness(t)
	h.s.Add(sc, 2, -12)
	h.run(30)
	h.want("samples", h.samples, "2@0", "2@8")
	h.want("stopped", h.stopped, "2@12")
}

func TestAddScript(t *testing.T) {
	sc := newTestScript(
		ins(opPlaySample, 1),
		ins(opUpdate),
		ins(opTag, 1),
		ins(opGotoTag, 1), // ignored
		ins(opPlaySample, 2),
		ins(opPurge), // ignored
		ins(opUpdate),
		ins(opTag, 2),
		ins(opPlaySample, 3),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.s.AddScript(sc)
	h.run(30)
	h.want("samples", h.samples, "1@0", "2@4", "3@8")
	h.want("stopped", h.stopped, "0@12")
}

func TestAddTwice(t *testing.T) {
	sc := newTestScript(ins(opTag, 1), ins(opUpdate), ins(opPurge), ins(opUpdate), ins(opTag, 2), ins(opUpdate))
	h := newHarness(t)
	if h.s.Add(sc, 1, 0) == nil {
		t.Fatal("first Add failed")
	}
	if h.s.Add(sc, 1, 0) != nil {
		t.Error("clip started twice")
	}
	if h.s.Add(sc, 2, 0) == nil {
		t.Error("other clip of the same script refused")
	}
	h.run(5) // clip 1 has reached PURGE and is ending
	if h.s.Add(sc, 1, 0) == nil {
		t.Error("ending clip can't be started again")
	}
}

// A new thread takes the lowest free slot, so it can be drawn below older
// ones, as in the original.
func TestSlots(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1), ins(opSetDelay, 4), ins(opUpdate), ins(opPurge), ins(opUpdate),
		ins(opTag, 2), ins(opSetDelay, 100), ins(opUpdate),
		ins(opTag, 3), ins(opSetDelay, 100), ins(opUpdate),
	)
	h := newHarness(t)
	h.s.Add(sc, 1, 0)
	h.s.Add(sc, 2, 0)
	h.run(10)
	t3 := h.s.Add(sc, 3, 0)
	if t3 == nil || t3.slot != 0 {
		t.Fatalf("new thread in slot %v, want 0", t3)
	}
	var tags []uint16
	for _, th := range h.s.Threads() {
		tags = append(tags, th.Tag)
	}
	if !slices.Equal(tags, []uint16{3, 2}) {
		t.Errorf("threads in slot order: %v, want [3 2]", tags)
	}

	full := newHarness(t)
	for i := range MaxThreads + 1 {
		full.s.Add(newTestScript(ins(opTag, 1), ins(opUpdate)), 1, 0)
		if i == MaxThreads && len(full.logs) != 1 {
			t.Errorf("no warning when all slots are taken: %v", full.logs)
		}
	}
}

func TestWarnings(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(0x9990),
		ins(0x9990), // same unknown opcode: not reported again
		ins(0x9981, 5),
		ins(opSetColors, 207, 3),
		ins(opDrawSprite, 0, 0, 0, 0),
		ins(opDrawCircle, 0, 0, 4, 6),
		ins(opGotoTag, 9),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.s.Add(sc, 1, 0)
	h.s.Add(sc, 7, 0)
	h.run(30)
	want := []string{
		"TEST.TTM: no tag 7",
		"TEST.TTM at 1: unknown opcode 0x9990, skipped here and from now on",
		"TEST.TTM at 3: unknown opcode 0x9981, skipped here and from now on",
		"TEST.TTM at 4: color 207 outside the palette, using 15",
		"TEST.TTM at 5: sprite 0 of bitmap slot 0 isn't loaded",
		"TEST.TTM at 6: can't draw a 4x6 circle",
		"TEST.TTM at 7: GOTO_TAG to missing tag 9",
	}
	h.want("logs", h.logs, want...)
}

// layerString draws the 8x4 corner of a Paletted layer as hex digits.
func layerString(pix []uint8) string {
	var b strings.Builder
	for y := range 4 {
		for x := range 8 {
			fmt.Fprintf(&b, "%x", pix[y*640+x])
		}
		b.WriteByte('/')
	}
	return b.String()
}

func TestDrawing(t *testing.T) {
	sprite := &resource.Image{Width: 3, Height: 2, Pix: []uint8{1, 0, 2, 3, 4, 5}}
	sc := newTestScript(
		ins(opTag, 1),
		ins(opSetBmpSlot, 2),
		str(opLoadImage, "S.BMP"),
		ins(opSetColors, 9, 9),
		ins(opDrawRect, 0, 0, 8, 4),
		ins(opSetColors, 0, 0),
		ins(opDrawPixel, 7, 3), // color 0 erases
		ins(opSetClipZone, 1, 0, 7, 3),
		ins(opDrawSprite, 0, 0, 0, 2),     // clipped on the left
		ins(opDrawSpriteFlip, 5, 1, 0, 2), // clipped at the bottom
		ins(opSetColors, 6, 6),
		ins(opDrawLine, 0, 2, 8, 2), // clipped at both ends
		ins(opPlaySample, 1),
		ins(opUpdate),
		ins(opCopyZoneToBG, 0, 0, 1, 1), // copies 3 columns
		ins(opClearScreen, 0),           // whole layer, despite the clip
		ins(opDrawRect, 0, 3, 8, 1),     // the clip is still set
		ins(opPlaySample, 2),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.p.bitmaps["S.BMP"] = []Sprite{sprite}
	var frames []string
	h.s.Sound = func(int) {
		frames = append(frames, layerString(h.p.layers[0].pix)+" saved "+layerString(h.p.saved.pix))
	}
	h.s.Add(sc, 1, 0)
	h.run(5)
	want := []string{
		"99299999/94599299/96666669/99999990/ saved 00000000/00000000/00000000/00000000/",
		"00000000/00000000/00000000/00000000/ saved 99200000/00000000/00000000/00000000/",
	}
	if !slices.Equal(frames, want) {
		t.Errorf("got  %q\nwant %q", frames, want)
	}
}

func TestOrigin(t *testing.T) {
	sc := newTestScript(
		ins(opTag, 1),
		ins(opSetClipZone, 0, 0, 3, 2),
		ins(opSetColors, 5, 5),
		ins(opDrawRect, 0, 0, 10, 10),
		ins(opUpdate),
	)
	h := newHarness(t)
	h.p.Origin.X, h.p.Origin.Y = 2, 1
	h.s.Add(sc, 1, 0)
	h.run(1)
	if got, want := layerString(h.p.layers[0].pix), "00000000/00555000/00555000/00000000/"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestCompose(t *testing.T) {
	p := NewPaletted(nil, testPalette())
	p.SetBackground(&resource.Image{Width: 2, Height: 1, Pix: []uint8{3, 4}})
	l := p.Layer(1)
	l.SetVisible(true)
	l.FillRect(1, 0, 1, 2, 7)
	p.saved.pix[0] = 2
	img := p.Compose()
	gray := func(x, y int) uint8 { return img.Pix[img.PixOffset(x, y)] }
	// Row 0: saved zone over the background, then the layer. Row 1 has no
	// background, so it is black except where the layer covers it.
	if got := []uint8{gray(0, 0), gray(1, 0), gray(2, 0), gray(0, 1), gray(1, 1)}; !slices.Equal(got, []uint8{12, 17, 10, 0, 17}) {
		t.Errorf("got %v", got)
	}
	l.SetVisible(false)
	if p.Compose().Pix[img.PixOffset(1, 1)] != 0 {
		t.Error("hidden layer still shown")
	}
}
