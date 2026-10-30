package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/xcifer/johnny-castaway-hd/internal/render"
)

// settings are the choices made in the settings dialog, kept in
// %APPDATA%\JohnnyCastawayHD\settings.json. Options given as flags win.
type settings struct {
	Mute     bool   `json:"mute,omitempty"`
	Filter   string `json:"filter,omitempty"`   // as -filter; "": sharp
	DataDir  string `json:"dataDir,omitempty"`  // "": searched, see dataDirs
	MainOnly bool   `json:"mainOnly,omitempty"` // Johnny on the main monitor only
}

// filters names the scaling filters for -filter and the settings.
var filters = map[string]render.Filter{
	"sharp":   render.SharpBilinear,
	"nearest": render.Nearest,
	"smooth":  render.Smooth,
}

// configFile is where a file of ours is kept: in
// %APPDATA%\JohnnyCastawayHD on Windows.
func configFile(name string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "JohnnyCastawayHD", name), nil
}

// loadSettings reads the settings. Problems are logged, and what can't be
// read is left as it is by default.
func loadSettings() settings {
	var s settings
	path, err := configFile("settings.json")
	if err == nil {
		var b []byte
		b, err = os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			return s
		} else if err == nil {
			err = json.Unmarshal(b, &s)
		}
	}
	if err != nil {
		log.Printf("settings: %v", err)
		return settings{}
	}
	if _, ok := filters[s.Filter]; !ok && s.Filter != "" {
		log.Printf("settings: unknown filter %q", s.Filter)
		s.Filter = ""
	}
	return s
}

func saveSettings(s settings) error {
	path, err := configFile("settings.json")
	if err != nil {
		return err
	}
	return writeConfig(path, s)
}

// writeConfig saves v as JSON. It writes a new file and renames it, so the
// old one stays whole if writing fails.
func writeConfig(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".new"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
