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

// mouseWatch tells when the mouse has been moved on purpose.
type mouseWatch struct {
	seen int
	rest image.Point
}

// moved reports whether p is further than slack pixels from where the
// mouse rested.
func (m *mouseWatch) moved(p image.Point, slack float64) bool {
	if m.seen < mouseSettle {
		m.seen++
		m.rest = p
		return false
	}
	d := p.Sub(m.rest)
	return math.Hypot(float64(d.X), float64(d.Y)) > slack
}
