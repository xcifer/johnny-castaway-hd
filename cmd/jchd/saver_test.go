package main

import (
	"image"
	"strings"
	"testing"
)

func TestParseSaverArgs(t *testing.T) {
	tests := []struct {
		args string
		mode saverMode
		hwnd uintptr
		bad  bool
	}{
		{"", saverSettings, 0, false},
		{"/s", saverRun, 0, false},
		{"/S", saverRun, 0, false},
		{"/s -windowed", saverRun, 0, false},
		{"/s 1234", saverRun, 0, false},
		{"/s:1234", saverRun, 0, false},
		{"/c", saverSettings, 0, false},
		{"/C", saverSettings, 0, false},
		{"/c:1234", saverSettings, 1234, false},
		{"/C:65802", saverSettings, 65802, false},
		{"/c 1234", saverSettings, 1234, false},
		{"/p 1234", saverPreview, 1234, false},
		{"/P 1234", saverPreview, 1234, false},
		{"/p:1234", saverPreview, 1234, false},
		{"/P:4294967295", saverPreview, 4294967295, false},
		{"-windowed", saverNone, 0, false},
		{"-windowed -debug", saverNone, 0, false},
		{"-data /games/jc", saverNone, 0, false},
		{"/p", saverNone, 0, true},
		{"/p:0", saverNone, 0, true},
		{"/p x", saverNone, 0, true},
		{"/c:-5", saverNone, 0, true},
		{"/sx", saverNone, 0, true},
		{"/a", saverNone, 0, true},
		{"/x:1", saverNone, 0, true},
		{"/", saverNone, 0, false},
	}
	for _, tt := range tests {
		mode, hwnd, err := parseSaverArgs(strings.Fields(tt.args))
		if mode != tt.mode || hwnd != tt.hwnd || (err != nil) != tt.bad {
			t.Errorf("parseSaverArgs(%q) = %d, %d, %v; want %d, %d, error %v", tt.args, mode, hwnd, err, tt.mode, tt.hwnd, tt.bad)
		}
	}
}

func TestOtherScreenArgs(t *testing.T) {
	got := strings.Join(otherScreenArgs(1, 4242, 7, false), " ")
	if want := "-monitor 1 -follow 4242 -mute -day 7"; got != want {
		t.Errorf("Johnny on every monitor: %q, want %q", got, want)
	}
	got = strings.Join(otherScreenArgs(2, 4242, 7, true), " ")
	if want := "-monitor 2 -follow 4242 -blank"; got != want {
		t.Errorf("main monitor only: %q, want %q", got, want)
	}
	// The copies must be started with options, never as Windows would.
	if mode, _, err := parseSaverArgs(otherScreenArgs(1, 4242, 7, false)); mode != saverNone || err != nil {
		t.Errorf("copy's arguments read as screensaver mode %d, %v", mode, err)
	}
}

func TestInputWatchMouse(t *testing.T) {
	var w inputWatch
	at := func(p image.Point) inputState { return inputState{mouse: p} }
	// While settling, the resting place follows the mouse, however far.
	for i := range mouseSettle {
		if w.used(at(image.Pt(i*50, 0)), 8) {
			t.Fatalf("used during settling, at %d", i)
		}
	}
	rest := image.Pt((mouseSettle-1)*50, 0)
	for _, d := range []image.Point{{0, 0}, {8, 0}, {0, -8}, {5, 6}, {-5, -6}, {-8, 0}} {
		if w.used(at(rest.Add(d)), 8) {
			t.Errorf("move by %v counted, slack 8", d)
		}
	}
	for _, d := range []image.Point{{9, 0}, {0, 9}, {6, 6}, {-7, -5}, {500, 300}} {
		if !w.used(at(rest.Add(d)), 8) {
			t.Errorf("move by %v not counted, slack 8", d)
		}
	}
	// The resting place doesn't creep along with small moves.
	if w.used(at(rest.Add(image.Pt(5, 0))), 8) || !w.used(at(rest.Add(image.Pt(10, 0))), 8) {
		t.Error("resting place moved after settling")
	}
	// Slack scales, as for 200% display scaling.
	if w.used(at(rest.Add(image.Pt(16, 0))), 16) || !w.used(at(rest.Add(image.Pt(17, 0))), 16) {
		t.Error("slack 16 not kept")
	}
}

func TestInputWatchKeys(t *testing.T) {
	const shift, a, leftButton = 0x10, 0x41, 0x01
	var w inputWatch
	in := inputState{}
	in.down[shift] = true // held from before the start
	if w.used(in, 8) {
		t.Fatal("a key held at the start counted")
	}
	for range 3 * mouseSettle {
		if w.used(in, 8) {
			t.Fatal("a key kept held counted")
		}
	}
	in.down[a] = true
	if !w.used(in, 8) {
		t.Error("a new key press not counted")
	}
	if w.used(in, 8) {
		t.Error("the same press counted twice")
	}
	in.down[shift], in.down[a] = false, false
	if w.used(in, 8) {
		t.Error("letting go of keys counted")
	}
	in.down[shift] = true
	if !w.used(in, 8) {
		t.Error("pressing the start's key again not counted")
	}

	// A press right after the start counts even while the mouse settles.
	var v inputWatch
	v.used(inputState{}, 8)
	click := inputState{}
	click.down[leftButton] = true
	if !v.used(click, 8) {
		t.Error("a click while settling not counted")
	}
}
