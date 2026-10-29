package script

// The TTM interpreter follows ttm.c and the thread handling in ads.c of
// jc_reborn by Jeremie Guillaume (GPL-3.0). docs/ttm-opcodes.md lists what
// each opcode does here and how sure we are of it.

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// TTM opcodes. The low nibble is the argument count (0xF: one string).
const (
	opDrawBackground = 0x0080
	opPurge          = 0x0110
	opUpdate         = 0x0FF0
	opSetDelay       = 0x1021
	opSetBmpSlot     = 0x1051
	opSetPaletteSlot = 0x1061
	opLocalTag       = 0x1101
	opTag            = 0x1111
	opUnknown1       = 0x1121
	opGotoTag        = 0x1201
	opSetColors      = 0x2002
	opSetFrame1      = 0x2012
	opTimer          = 0x2022
	opSetClipZone    = 0x4004
	opCopyZoneToBG   = 0x4204
	opSaveImage1     = 0x4214
	opDrawPixel      = 0xA002
	opSaveZone       = 0xA054
	opRestoreZone    = 0xA064
	opDrawLine       = 0xA0A4
	opDrawRect       = 0xA104
	opDrawCircle     = 0xA404
	opDrawSprite     = 0xA504
	opDrawSpriteFlip = 0xA524
	opClearScreen    = 0xA601
	opDrawScreen     = 0xB606
	opPlaySample     = 0xC051
	opLoadScreen     = 0xF01F
	opLoadImage      = 0xF02F
	opLoadPalette    = 0xF05F
)

const (
	// MaxThreads is how many clips can play at once.
	MaxThreads = 10
	bmpSlots   = 6

	// TicksPerSecond is the clock script delays count in: 1/64 s.
	TicksPerSecond = 64
	// FrameTicks is how often the original redraws the screen: every 4
	// ticks (62.5 ms). A frame stays up for its delay rounded up to whole
	// redraws, and for at least one. Measured in recordings of the
	// original; see docs/timing.md.
	FrameTicks = 4
	// startDelay is the delay of a clip that doesn't set one.
	startDelay = 4
)

// Script is a loaded TTM script: numbered clips, each running from a tag to
// a PURGE. Its bitmap slots are shared by every thread playing it, so one
// clip can load the pictures another one draws.
type Script struct {
	Name    string
	code    []resource.Instr
	tags    []tagPos // in code order
	bitmaps [bmpSlots][]Sprite
}

type tagPos struct {
	id uint16
	ip int // index of the instruction after the tag
}

// NewScript prepares a decoded TTM for playing.
func NewScript(name string, t *resource.TTM) *Script {
	s := &Script{Name: name, code: t.Code}
	for i, in := range t.Code {
		if in.Opcode == opTag || in.Opcode == opLocalTag {
			s.tags = append(s.tags, tagPos{in.Args[0], i + 1})
		}
	}
	return s
}

// Tags lists the script's tag numbers in code order.
func (s *Script) Tags() []uint16 {
	ids := make([]uint16, len(s.tags))
	for i, t := range s.tags {
		ids[i] = t.id
	}
	return ids
}

// findTag returns where the first tag numbered id starts.
func (s *Script) findTag(id uint16) (int, bool) {
	for _, t := range s.tags {
		if t.id == id {
			return t.ip, true
		}
	}
	return 0, false
}

// tagBefore returns the start of the clip ip is in: the last tag that
// starts before it.
func (s *Script) tagBefore(ip int) (int, bool) {
	start, ok := 0, false
	for _, t := range s.tags {
		if t.ip >= ip {
			break
		}
		start, ok = t.ip, true
	}
	return start, ok
}

// Thread plays a clip of a script on its own layer.
type Thread struct {
	Script *Script
	Tag    uint16
	slot   int
	layer  Layer

	ip         int
	delay      int  // ticks the frame should stay up
	timer      int  // ticks left of the frame on screen
	age        int  // ticks the frame on screen has been up
	shown      bool // a frame is on screen; false until the first one
	jump       int  // where to continue after this frame, or -1
	bmp        int
	fg, bg     uint8
	sceneTimer int // ticks left to keep looping the clip, or 0
	repeats    int // times left to replay the clip once it ends
	whole      bool
	ending     bool
}

func (t *Thread) String() string {
	if t.whole {
		return fmt.Sprintf("%s ip %d", t.Script.Name, t.ip)
	}
	return fmt.Sprintf("%s:%d ip %d", t.Script.Name, t.Tag, t.ip)
}

// Scheduler runs threads in slots. Every FrameTicks ticks, call Step and
// then Play.
type Scheduler struct {
	Canvas Canvas
	// Sound plays a sample for PLAY_SAMPLE; nil ignores them.
	Sound func(n int)
	// Rand picks the random delays of TIMER; nil uses the global source.
	Rand *rand.Rand
	// Logf reports problems in scripts, each one only once; nil uses the
	// log package.
	Logf func(format string, args ...any)

	threads [MaxThreads]*Thread
	warned  map[string]bool
	unknown map[uint16]bool
}

// NewScheduler returns a scheduler drawing on c.
func NewScheduler(c Canvas) *Scheduler {
	return &Scheduler{Canvas: c, warned: map[string]bool{}, unknown: map[uint16]bool{}}
}

func (s *Scheduler) warnf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if s.warned[msg] {
		return
	}
	s.warned[msg] = true
	if s.Logf != nil {
		s.Logf("%s", msg)
	} else {
		log.Print(msg)
	}
}

// Add starts the clip of sc at tag, as a scene script's ADD_SCENE does. If
// n > 0 the clip plays n times; if n < 0 it loops for -n ticks. A clip that
// is already playing isn't started again. It returns the new thread, or nil.
func (s *Scheduler) Add(sc *Script, tag uint16, n int) *Thread {
	for _, t := range s.threads {
		if t != nil && !t.ending && t.Script == sc && t.Tag == tag {
			return nil
		}
	}
	ip, ok := sc.findTag(tag)
	if !ok {
		s.warnf("%s: no tag %d", sc.Name, tag)
	}
	t := s.start(sc, ip)
	if t == nil {
		return nil
	}
	t.Tag = tag
	if n < 0 {
		t.sceneTimer = -n
	} else if n > 0 {
		t.repeats = n - 1
	}
	return t
}

// AddScript plays the whole of sc from the start, through every clip, until
// its code runs out. Clip ends and jumps are ignored. It is for watching a
// script on its own.
func (s *Scheduler) AddScript(sc *Script) *Thread {
	t := s.start(sc, 0)
	if t != nil {
		t.whole = true
	}
	return t
}

func (s *Scheduler) start(sc *Script, ip int) *Thread {
	for i, t := range s.threads {
		if t != nil {
			continue
		}
		l := s.Canvas.Layer(i)
		l.Clear()
		l.ResetClip()
		l.SetVisible(true)
		t = &Thread{Script: sc, slot: i, layer: l, ip: ip, delay: startDelay, jump: -1, fg: 15, bg: 15}
		s.threads[i] = t
		return t
	}
	s.warnf("%s: more than %d clips at once", sc.Name, MaxThreads)
	return nil
}

func (s *Scheduler) stop(t *Thread) {
	t.layer.SetVisible(false)
	s.threads[t.slot] = nil
}

// Threads returns the playing threads in slot order.
func (s *Scheduler) Threads() []*Thread {
	var ts []*Thread
	for _, t := range s.threads {
		if t != nil {
			ts = append(ts, t)
		}
	}
	return ts
}

// Step advances the clock by FrameTicks ticks. Threads whose frame is over
// move on: a pending jump is taken, a timed clip runs down, and an ended
// clip is replayed or stopped. Step returns the threads that stopped.
func (s *Scheduler) Step() []*Thread {
	var stopped []*Thread
	for _, t := range s.threads {
		if t == nil || !t.shown {
			continue
		}
		t.timer -= FrameTicks
		t.age += FrameTicks
		if t.timer > 0 {
			continue
		}
		t.shown = false
		if t.whole {
			t.jump = -1
			t.ending = t.ip >= len(t.Script.code)
		}
		if t.jump >= 0 {
			t.ip, t.jump = t.jump, -1
		}
		if t.sceneTimer > 0 {
			t.sceneTimer -= t.age
			if t.sceneTimer <= 0 {
				t.ending = true
			}
		}
		if t.ending {
			if t.repeats > 0 {
				t.repeats--
				t.ending = false
				t.ip, _ = t.Script.findTag(t.Tag)
			} else {
				s.stop(t)
				stopped = append(stopped, t)
			}
		}
	}
	return stopped
}

// Play draws the next frame of every thread whose turn it is: it runs
// each one's code up to the next UPDATE.
func (s *Scheduler) Play() {
	for _, t := range s.threads {
		if t != nil && !t.shown {
			s.run(t)
			t.timer, t.age, t.shown = t.delay, 0, true
		}
	}
}

func (s *Scheduler) run(t *Thread) {
	code := t.Script.code
	for t.ip < len(code) {
		in := code[t.ip]
		t.ip++
		update := s.exec(t, in)
		if t.ip >= len(code) {
			break
		}
		if update {
			return
		}
	}
	t.ending = true
}

func (s *Scheduler) intN(n int) int {
	if s.Rand != nil {
		return s.Rand.IntN(n)
	}
	return rand.IntN(n)
}

// coord reads a coordinate argument, which may be negative.
func coord(v uint16) int { return int(int16(v)) }

// color reads a color argument. SASKDATE.TTM once uses 207 (0xCF), maybe a
// 256-color index; in 16 colors only the low four bits can count.
func (s *Scheduler) color(t *Thread, in resource.Instr, v uint16) uint8 {
	if v > 15 {
		s.warnf("%s at %d: color %d outside the palette, using %d", t.Script.Name, in.Offset, v, v&15)
	}
	return uint8(v & 15)
}

// exec runs one instruction and reports whether it ends the frame.
func (s *Scheduler) exec(t *Thread, in resource.Instr) bool {
	a := in.Args
	switch in.Opcode {
	case opUpdate:
		return true
	case opPurge:
		if t.sceneTimer != 0 {
			if ip, ok := t.Script.tagBefore(t.ip); ok {
				t.jump = ip
			}
		} else {
			t.ending = true
		}
	case opSetDelay:
		t.delay = int(a[0])
	case opTimer:
		// A random delay from a to b: the same TIMER waits differently
		// each time in recordings of the original.
		lo, hi := int(a[0]), max(int(a[1]), int(a[0]))
		t.delay = lo + s.intN(hi-lo+1)
	case opSetBmpSlot:
		t.bmp = int(a[0])
	case opGotoTag:
		if ip, ok := t.Script.findTag(a[0]); ok {
			t.jump = ip
		} else {
			s.warnf("%s at %d: GOTO_TAG to missing tag %d", t.Script.Name, in.Offset, a[0])
		}
	case opSetColors:
		t.fg, t.bg = s.color(t, in, a[0]), s.color(t, in, a[1])
	case opSetClipZone:
		t.layer.SetClip(coord(a[0]), coord(a[1]), coord(a[2]), coord(a[3]))
	case opCopyZoneToBG:
		// Two more columns than asked for, as in jc_reborn: GJVIS6.TTM
		// leaves a two-pixel gap in the cargo ship's hull otherwise.
		s.Canvas.SaveZone(t.layer, coord(a[0]), coord(a[1]), int(a[2])+2, int(a[3]))
	case opRestoreZone:
		// Only GJGULIVR.TTM saves and restores a zone, once, with no other
		// saved zone around, so restoring can simply clear them all.
		s.Canvas.ClearSaved()
	case opDrawPixel:
		t.layer.FillRect(coord(a[0]), coord(a[1]), 1, 1, t.fg)
	case opDrawLine:
		drawLine(t.layer, coord(a[0]), coord(a[1]), coord(a[2]), coord(a[3]), t.fg)
	case opDrawRect:
		t.layer.FillRect(coord(a[0]), coord(a[1]), int(a[2]), int(a[3]), t.fg)
	case opDrawCircle:
		if !drawCircle(t.layer, coord(a[0]), coord(a[1]), int(a[2]), int(a[3]), t.fg, t.bg) {
			s.warnf("%s at %d: can't draw a %dx%d circle", t.Script.Name, in.Offset, a[2], a[3])
		}
	case opDrawSprite, opDrawSpriteFlip:
		s.drawSprite(t, in, in.Opcode == opDrawSpriteFlip)
	case opClearScreen:
		t.layer.Clear()
	case opPlaySample:
		if s.Sound != nil {
			s.Sound(int(a[0]))
		}
	case opLoadScreen:
		if err := s.Canvas.LoadScreen(in.Str); err != nil {
			s.warnf("%s at %d: %v", t.Script.Name, in.Offset, err)
		}
	case opLoadImage:
		if t.bmp >= bmpSlots {
			s.warnf("%s at %d: bitmap slot %d out of range", t.Script.Name, in.Offset, t.bmp)
			break
		}
		sprites, err := s.Canvas.LoadBitmap(in.Str)
		if err != nil {
			s.warnf("%s at %d: %v", t.Script.Name, in.Offset, err)
		}
		t.Script.bitmaps[t.bmp] = sprites
	case opDrawBackground, opSetPaletteSlot, opLocalTag, opTag, opUnknown1,
		opSetFrame1, opSaveImage1, opSaveZone, opDrawScreen, opLoadPalette:
		// Not needed to draw the game; see docs/ttm-opcodes.md.
	default:
		if !s.unknown[in.Opcode] {
			s.unknown[in.Opcode] = true
			s.warnf("%s at %d: unknown opcode 0x%04X, skipped here and from now on", t.Script.Name, in.Offset, in.Opcode)
		}
	}
	return false
}

func (s *Scheduler) drawSprite(t *Thread, in resource.Instr, flip bool) {
	n, slot := int(in.Args[2]), int(in.Args[3])
	if slot >= bmpSlots || n >= len(t.Script.bitmaps[slot]) {
		s.warnf("%s at %d: sprite %d of bitmap slot %d isn't loaded", t.Script.Name, in.Offset, n, slot)
		return
	}
	t.layer.DrawSprite(t.Script.bitmaps[slot][n], coord(in.Args[0]), coord(in.Args[1]), flip)
}
