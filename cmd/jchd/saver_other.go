//go:build !windows

package main

import (
	"image"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// showSettings shows the settings. Only Windows has a settings dialog.
func showSettings(parent uintptr) {
	log.Print("Johnny Castaway HD has its settings in settings.json.")
}

// showPreview draws into the preview window. Only Windows asks for one.
func showPreview(parent uintptr) {}

// readInput reads the mouse and keyboard as the window hears of them. Keys
// are numbered by Ebitengine here, and mouse buttons follow after them.
func readInput() (inputState, bool) {
	var in inputState
	x, y := ebiten.CursorPosition()
	in.mouse = image.Pt(x, y)
	for k := ebiten.Key(0); k <= ebiten.KeyMax && int(k) < 240; k++ {
		in.down[k] = ebiten.IsKeyPressed(k)
	}
	for b := ebiten.MouseButton(0); b <= ebiten.MouseButtonMax; b++ {
		in.down[240+int(b)] = ebiten.IsMouseButtonPressed(b)
	}
	return in, true
}

// waitProcess waits for the process pid to end, checking every second.
func waitProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	for p.Signal(syscall.Signal(0)) == nil {
		time.Sleep(time.Second)
	}
	return nil
}
