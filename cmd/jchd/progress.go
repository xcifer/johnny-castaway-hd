package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/xcifer/johnny-castaway-hd/internal/story"
)

// progressFile is where the story's progress is kept between runs:
// %APPDATA%\JohnnyCastawayHD\progress.json on Windows.
func progressFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "JohnnyCastawayHD", "progress.json"), nil
}

// savedDay returns a function telling the day of the story, which moves on
// a day on each new date and is saved. Problems with the file are logged,
// and the story then goes on from day 1 without saving.
func savedDay(now func() time.Time) func() int {
	var p story.Progress
	path, err := progressFile()
	if err == nil {
		var b []byte
		b, err = os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		} else if err == nil {
			err = json.Unmarshal(b, &p)
		}
	}
	if err != nil {
		log.Printf("story progress: %v", err)
		path = ""
	}
	return func() int {
		if p.Advance(now()) && path != "" {
			if err := saveProgress(path, p); err != nil {
				log.Printf("story progress: %v", err)
				path = ""
			}
		}
		return p.Day
	}
}

func saveProgress(path string, p story.Progress) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Write a new file and rename it, so the old one stays whole if
	// writing fails.
	tmp := path + ".new"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
