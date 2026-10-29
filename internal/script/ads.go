package script

// The scene interpreter follows ads.c of jc_reborn by Jeremie Guillaume
// (GPL-3.0). docs/ads-opcodes.md lists what each opcode does here.

import (
	"fmt"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// ADS opcodes. Any other value in the code is a tag label.
const (
	adsIfLastPlayedLocal = 0x1070
	adsIfUnknown1        = 0x1330
	adsIfLastPlayed      = 0x1350
	adsIfNotRunning      = 0x1360
	adsIfIsRunning       = 0x1370
	adsAnd               = 0x1420
	adsOr                = 0x1430
	adsPlayScene         = 0x1510
	adsAddSceneLocal     = 0x1520
	adsAddScene          = 0x2005
	adsStopScene         = 0x2010
	adsRandomStart       = 0x3010
	adsNop               = 0x3020
	adsRandomEnd         = 0x30FF
	adsUnknown6          = 0x4000
	adsFadeOut           = 0xF010
	adsGosubTag          = 0xF200
	adsEnd               = 0xFFFF
	adsEndIf             = 0xFFF0
)

// clipRef names a clip: a script slot of the scene and a tag in it.
type clipRef struct{ slot, tag uint16 }

// trigger is code to run when a clip ends.
type trigger struct {
	clip clipRef
	ip   int
}

// choice is one option of a RANDOM_START ... RANDOM_END block.
type choice struct {
	op     uint16 // adsAddScene, adsStopScene or adsNop
	clip   clipRef
	n      int
	weight int
}

// Scene plays one scene of an ADS script: a tag whose code starts clips,
// and blocks that start more clips when others end. It is over when no
// clip plays.
type Scene struct {
	Name string
	Tag  uint16

	sched   *Scheduler
	code    []resource.Instr
	tags    map[uint16]int // tag label -> index of the instruction after it
	scripts map[uint16]*Script
	slots   map[*Script]uint16

	triggers []trigger
	local    []trigger // IF_LASTPLAYED_LOCAL, which replaces the others
	choices  []choice
	stopping bool // END was reached: ending clips trigger nothing more
}

// NewScene prepares scene tag of ads, with its scripts already loaded by
// slot, and runs the scene's first block, which starts its first clips.
// Call Step on the scene instead of the scheduler's.
func NewScene(s *Scheduler, name string, ads *resource.ADS, scripts map[uint16]*Script, tag uint16) (*Scene, error) {
	sc := &Scene{Name: name, Tag: tag, sched: s, code: ads.Code, tags: map[uint16]int{},
		scripts: scripts, slots: map[*Script]uint16{}}
	for slot, script := range scripts {
		sc.slots[script] = slot
	}

	// Bookmark the scene's triggers: each IF_LASTPLAYED after the scene's
	// tag, up to the next tag, and the IF_NOT_RUNNINGs before the first
	// IF_LASTPLAYED or IF_IS_RUNNING.
	start := -1
	inScene, leading := false, false
	for i, in := range ads.Code {
		switch in.Opcode {
		case adsIfLastPlayed:
			if inScene {
				leading = false
				sc.triggers = append(sc.triggers, trigger{clipRef{in.Args[0], in.Args[1]}, i + 1})
			}
		case adsIfNotRunning:
			if inScene && leading {
				sc.triggers = append(sc.triggers, trigger{clipRef{in.Args[0], in.Args[1]}, i + 1})
			}
		case adsIfIsRunning:
			leading = false
		default:
			if resource.IsADSOpcode(in.Opcode) {
				continue
			}
			sc.tags[in.Opcode] = i + 1
			inScene = in.Opcode == tag
			leading = inScene
			if inScene && start < 0 {
				start = i + 1
			}
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("%s has no scene %d", name, tag)
	}
	sc.run(start)
	return sc, nil
}

// Step advances the scene by one redraw, as Scheduler.Step does, and runs
// the blocks waiting for the clips that ended.
func (sc *Scene) Step() {
	for _, t := range sc.sched.Step() {
		if sc.stopping {
			continue
		}
		slot, ok := sc.slots[t.Script]
		if !ok {
			continue
		}
		sc.clipEnded(clipRef{slot, t.Tag})
	}
}

// Done reports whether the scene is over.
func (sc *Scene) Done() bool { return len(sc.sched.Threads()) == 0 }

func (sc *Scene) clipEnded(c clipRef) {
	// The one local trigger (ACTIVITY.ADS tag 7) replaces the scene's
	// triggers while it waits, and fires once.
	if len(sc.local) > 0 {
		for i := 0; i < len(sc.local); i++ {
			if sc.local[i].clip == c {
				ip := sc.local[i].ip
				sc.local = append(sc.local[:i], sc.local[i+1:]...)
				i--
				sc.run(ip)
			}
		}
		return
	}
	// A clip can have several triggers (BUILDING.ADS tag 2).
	for _, tr := range sc.triggers {
		if tr.clip == c {
			sc.run(tr.ip)
		}
	}
}

// running reports whether a clip plays and isn't ending.
func (sc *Scene) running(c clipRef) bool {
	script := sc.scripts[c.slot]
	for _, t := range sc.sched.threads {
		if t != nil && !t.ending && t.Script == script && t.Tag == c.tag {
			return true
		}
	}
	return false
}

func (sc *Scene) add(c clipRef, n int) {
	script := sc.scripts[c.slot]
	if script == nil {
		sc.sched.warnf("%s: no script in slot %d", sc.Name, c.slot)
		return
	}
	sc.sched.Add(script, c.tag, n)
}

func (sc *Scene) stop(c clipRef) {
	script := sc.scripts[c.slot]
	for _, t := range sc.sched.threads {
		if t != nil && t.Script == script && t.Tag == c.tag {
			sc.sched.stop(t)
		}
	}
}

// run runs a block of scene code from instruction ip. A block ends at
// PLAY_SCENE (outside a skipped part) or at the next IF_LASTPLAYED that
// doesn't follow an OR.
func (sc *Scene) run(ip int) {
	inRandom, inOr, skip, localPending := false, false, false, false
	for ; ip < len(sc.code); ip++ {
		in := sc.code[ip]
		a := in.Args
		switch in.Opcode {
		case adsIfLastPlayedLocal:
			sc.local = append(sc.local, trigger{clipRef{a[0], a[1]}, ip + 1})
			localPending = true
		case adsIfLastPlayed:
			if !inOr {
				return
			}
			inOr = false
		case adsIfNotRunning:
			if sc.running(clipRef{a[0], a[1]}) {
				skip = true
			}
		case adsIfIsRunning:
			skip = !sc.running(clipRef{a[0], a[1]})
		case adsOr:
			inOr = true
		case adsPlayScene:
			if !skip {
				return
			}
			skip = false
		case adsAddSceneLocal:
			// Its five words are 0x2005 and the arguments of an ordinary
			// ADD_SCENE. Reached while queuing the local trigger it does
			// nothing; reached from the trigger it starts the clip.
			if localPending {
				localPending = false
			} else {
				sc.add(clipRef{a[1], a[2]}, int(int16(a[3])))
			}
		case adsAddScene:
			if skip {
				break
			}
			c := clipRef{a[0], a[1]}
			if inRandom {
				sc.choices = append(sc.choices, choice{adsAddScene, c, int(int16(a[2])), int(a[3])})
			} else {
				sc.add(c, int(int16(a[2])))
			}
		case adsStopScene:
			if skip {
				break
			}
			c := clipRef{a[0], a[1]}
			if inRandom {
				sc.choices = append(sc.choices, choice{op: adsStopScene, clip: c, weight: int(a[2])})
			} else {
				sc.stop(c)
			}
		case adsRandomStart:
			sc.choices = sc.choices[:0]
			inRandom = true
		case adsNop:
			if inRandom {
				sc.choices = append(sc.choices, choice{op: adsNop, weight: int(a[0])})
			}
		case adsRandomEnd:
			sc.choose()
			inRandom = false
		case adsGosubTag:
			// Only STAND.ADS uses it, for a tag holding one block.
			if t, ok := sc.tags[a[0]]; ok {
				sc.run(t)
			} else {
				sc.sched.warnf("%s: GOSUB_TAG to missing tag %d", sc.Name, a[0])
			}
		case adsEnd:
			if skip {
				skip = false
			} else {
				sc.stopping = true
			}
		case adsIfUnknown1, adsAnd, adsUnknown6, adsFadeOut, adsEndIf:
			// See docs/ads-opcodes.md.
		}
	}
}

// choose carries out one option of a random block, picked by weight.
func (sc *Scene) choose() {
	total := 0
	for _, c := range sc.choices {
		total += c.weight
	}
	if total <= 0 {
		return
	}
	n := sc.sched.intN(total)
	for _, c := range sc.choices {
		if n -= c.weight; n < 0 {
			switch c.op {
			case adsAddScene:
				sc.add(c.clip, c.n)
			case adsStopScene:
				sc.stop(c.clip)
			}
			return
		}
	}
}
