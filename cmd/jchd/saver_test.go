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

func TestMouseWatch(t *testing.T) {
	var m mouseWatch
	// While settling, the resting place follows the mouse, however far.
	for i := range mouseSettle {
		if m.moved(image.Pt(i*50, 0), 8) {
			t.Fatalf("moved during settling, at %d", i)
		}
	}
	rest := image.Pt((mouseSettle-1)*50, 0)
	for _, d := range []image.Point{{0, 0}, {8, 0}, {0, -8}, {5, 6}, {-5, -6}, {-8, 0}} {
		if m.moved(rest.Add(d), 8) {
			t.Errorf("moved by %v counted, slack 8", d)
		}
	}
	for _, d := range []image.Point{{9, 0}, {0, 9}, {6, 6}, {-7, -5}, {500, 300}} {
		if !m.moved(rest.Add(d), 8) {
			t.Errorf("moved by %v not counted, slack 8", d)
		}
	}
	// The resting place doesn't creep along with small moves.
	if m.moved(rest.Add(image.Pt(5, 0)), 8) || !m.moved(rest.Add(image.Pt(10, 0)), 8) {
		t.Error("resting place moved after settling")
	}
	// Slack scales, as for 200% display scaling.
	if m.moved(rest.Add(image.Pt(16, 0)), 16) || !m.moved(rest.Add(image.Pt(17, 0)), 16) {
		t.Error("slack 16 not kept")
	}
}
