package main

import (
	"image"
	"os"
	"unsafe"
)

var (
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	procGetLastInputInfo = user32.NewProc("GetLastInputInfo")
)

// keys is the keyboard as last read, and lastInput the time of the last
// input then.
var (
	keys      [256]bool
	lastInput uint32
	keysRead  bool
)

// readInput reads the mouse and keyboard from Windows itself. The window
// only hears of them while it has the focus and the mouse is over it, and
// another window can keep both even with the screensaver in front.
func readInput() (inputState, bool) {
	var in inputState
	var p struct{ X, Y int32 }
	if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p))); r == 0 {
		return in, false // such as while the screen is locked
	}
	in.mouse = image.Pt(int(p.X), int(p.Y))
	// Reading every key takes a while, so only do it after new input.
	info := struct{ size, time uint32 }{size: 8}
	procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if !keysRead || info.time != lastInput {
		for k := 1; k < len(keys)-1; k++ {
			r, _, _ := procGetAsyncKeyState.Call(uintptr(k))
			keys[k] = r&0x8000 != 0
		}
		lastInput, keysRead = info.time, true
	}
	in.down = keys
	return in, true
}

// waitProcess waits for the process pid to end. On Windows it can be any
// process, not only a child.
func waitProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_, err = p.Wait()
	return err
}
