package main

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"
)

// saverMode is what Windows asks the screensaver to do.
type saverMode int

const (
	saverNone     saverMode = iota // not started by Windows: options as flags
	saverRun                       // /s: run fullscreen
	saverSettings                  // /c, or no arguments: show the settings
	saverPreview                   // /p HWND: draw into the preview window
)

// parseSaverArgs reads the arguments Windows starts a screensaver with: /s
// to run it, /c to show its settings (/c:HWND names the window to put them
// over) and /p HWND to draw a preview into that window. The letter may be
// upper or lower case, and the window may follow after a colon or as the
// next argument. With no arguments at all, Windows wants the settings.
// Arguments that start with a dash are options for development instead.
func parseSaverArgs(args []string) (saverMode, uintptr, error) {
	if len(args) == 0 {
		return saverSettings, 0, nil
	}
	a := args[0]
	if !strings.HasPrefix(a, "/") || len(a) < 2 {
		return saverNone, 0, nil
	}
	letter, rest := strings.ToLower(a[1:2]), a[2:]
	if letter == "s" && (rest == "" || rest[0] == ':') {
		return saverRun, 0, nil
	}
	if strings.HasPrefix(rest, ":") {
		rest = rest[1:]
	} else if rest == "" && len(args) > 1 {
		rest = args[1]
	} else if rest != "" {
		return saverNone, 0, fmt.Errorf("unknown screensaver argument %q", a)
	}
	var hwnd uint64
	if rest != "" {
		var err error
		if hwnd, err = strconv.ParseUint(rest, 10, 64); err != nil {
			return saverNone, 0, fmt.Errorf("bad window %q after %s", rest, a)
		}
	}
	switch letter {
	case "c":
		return saverSettings, uintptr(hwnd), nil
	case "p":
		if hwnd == 0 {
			return saverNone, 0, fmt.Errorf("%s needs a window", a)
		}
		return saverPreview, uintptr(hwnd), nil
	}
	return saverNone, 0, fmt.Errorf("unknown screensaver argument %q", a)
}

// mouseSettle is how many ticks after the start the mouse's resting place
// is still taken as reported: the first positions can be stale, and a hand
// can still be on the mouse after clicking Preview.
const mouseSettle = ticksPerSecond / 2

// mouseSlack is how far the mouse may move from its resting place before
// the screensaver ends, in pixels at 100% display scaling. Smaller moves,
// such as from a bumped desk, are ignored.
const mouseSlack = 8

// inputState is what the mouse and keyboard are doing: where the mouse is
// and which keys and mouse buttons are down, by Windows' virtual-key code.
type inputState struct {
	mouse image.Point
	down  [256]bool
}

// inputWatch tells when the mouse or keyboard has been used on purpose.
type inputWatch struct {
	seen int
	rest image.Point
	down [256]bool
}

// used reports whether, compared with what it saw before, a key or mouse
// button has gone down, or the mouse is further than slack pixels from
// where it rested. Keys already down at the start, such as one still held
// from clicking Preview, don't count until pressed again.
func (w *inputWatch) used(in inputState, slack float64) bool {
	pressed := false
	if w.seen > 0 {
		for k, d := range in.down {
			if d && !w.down[k] {
				pressed = true
			}
		}
	}
	w.down = in.down
	if w.seen < mouseSettle {
		w.seen++
		w.rest = in.mouse
		return pressed
	}
	d := in.mouse.Sub(w.rest)
	return pressed || math.Hypot(float64(d.X), float64(d.Y)) > slack
}
