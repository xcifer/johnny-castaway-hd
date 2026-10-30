package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/xcifer/johnny-castaway-hd/internal/story"
)

// savedDay returns a function telling the day of the story, which moves on
// a day on each new date and is saved in progress.json, next to the
// settings. Problems with the file are logged, and the story then goes on
// from day 1 without saving.
func savedDay(now func() time.Time) func() int {
	var p story.Progress
	path, err := configFile("progress.json")
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
			if err := writeConfig(path, p); err != nil {
				log.Printf("story progress: %v", err)
				path = ""
			}
		}
		return p.Day
	}
}
