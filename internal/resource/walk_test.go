package resource

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWalkPaths(t *testing.T) {
	steps := []WalkStep{{X: 1}, {X: 2}, {X: 0}, {X: 0}, {X: 3}, {X: 0}, {X: 4}}
	got := fmt.Sprint(WalkPaths(steps))
	want := "[[{false 0 1 0} {false 0 2 0}] [{false 0 3 0}] [{false 0 4 0}]]"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestWalkTableRejectsUnknownVersion(t *testing.T) {
	if _, err := WalkTable(make([]byte, 0x20000)); err == nil {
		t.Error("want error for unknown executable")
	}
}

func TestGameDataWalkTable(t *testing.T) {
	exe, err := os.ReadFile(filepath.Join(gameData, "SCRANTIC.SCR"))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	if err != nil {
		t.Fatal(err)
	}
	steps, err := WalkTable(exe)
	if err != nil {
		t.Fatal(err)
	}
	// Spot checks against jc_reborn's walk_data.h.
	for i, want := range map[int]WalkStep{
		0: {Flip: true, Sprite: 19, X: 306, Y: 238},
	} {
		if steps[i] != want {
			t.Errorf("step %d = %+v, want %+v", i, steps[i], want)
		}
	}
	found := false
	for _, s := range steps {
		if s == (WalkStep{Sprite: 4, X: 311, Y: 242}) { // first step of "A to F"
			found = true
		}
	}
	if !found {
		t.Error(`"A to F" start step not found`)
	}

	a := loadGameData(t)
	sprites, err := DecodeBMP(a.Find("JOHNWALK.BMP"))
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range steps {
		if s.X != 0 && (s.Sprite >= len(sprites.Frames) || s.X > 640 || s.Y > 480) {
			t.Errorf("step %d out of range: %+v", i, s)
		}
	}
	paths := WalkPaths(steps)
	if len(paths) == 0 {
		t.Fatal("no paths")
	}
	t.Logf("%d steps in %d paths", len(steps), len(paths))
}
