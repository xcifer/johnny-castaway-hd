package main

import (
	"os"
	"path/filepath"
	"testing"
)

// useConfigDir points os.UserConfigDir at a fresh folder for one test.
func useConfigDir(t *testing.T) string {
	dir := t.TempDir()
	t.Setenv("AppData", dir)         // Windows
	t.Setenv("XDG_CONFIG_HOME", dir) // others
	return filepath.Join(dir, "JohnnyCastawayHD")
}

func TestSettings(t *testing.T) {
	dir := useConfigDir(t)
	if s := loadSettings(); s != (settings{}) {
		t.Errorf("no file: got %+v", s)
	}

	want := settings{Mute: true, Filter: "smooth", DataDir: `D:\Johnny`, MainOnly: true}
	if err := saveSettings(want); err != nil {
		t.Fatal(err)
	}
	if s := loadSettings(); s != want {
		t.Errorf("after save: got %+v, want %+v", s, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.json.new")); !os.IsNotExist(err) {
		t.Errorf("temporary file left behind: %v", err)
	}

	write := func(text string) {
		if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"mute": true, "filter": "blurry"}`)
	if s := loadSettings(); s != (settings{Mute: true}) {
		t.Errorf("unknown filter: got %+v", s)
	}
	write(`{"mute": tru`)
	if s := loadSettings(); s != (settings{}) {
		t.Errorf("broken file: got %+v", s)
	}
	write(`{"filter": "nearest", "hdArt": true}`)
	if s := loadSettings(); s != (settings{Filter: "nearest"}) {
		t.Errorf("file with a newer setting: got %+v", s)
	}
}
