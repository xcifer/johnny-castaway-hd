package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// dataDirs lists where the game data is looked for when -data isn't given:
// next to the program, in a gamedata folder next to it, and in a gamedata
// folder in the current folder. Windows starts a screensaver in a folder of
// its own choosing, so the current folder alone won't do.
func dataDirs(exe, cwd string) []string {
	dir := filepath.Dir(exe)
	return []string{dir, filepath.Join(dir, "gamedata"), filepath.Join(cwd, "gamedata")}
}

// findDataDir returns the first of dirs that holds RESOURCE.MAP.
func findDataDir(dirs []string) (string, bool) {
	for _, d := range dirs {
		if fi, err := os.Stat(filepath.Join(d, "RESOURCE.MAP")); err == nil && fi.Mode().IsRegular() {
			return d, true
		}
	}
	return "", false
}

// defaultDataDir finds the game data in one of the dataDirs places.
func defaultDataDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if d, ok := findDataDir(dataDirs(exe, cwd)); ok {
		return d, nil
	}
	return "", fmt.Errorf("put RESOURCE.MAP, RESOURCE.001 and SCRANTIC.SCR\nfrom the original next to %s,\nor in a folder called gamedata there", filepath.Base(exe))
}
