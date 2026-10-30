package main

import "golang.org/x/sys/windows"

// showSettings shows the settings over the window parent (0: none).
func showSettings(parent uintptr) {
	title, _ := windows.UTF16PtrFromString("Johnny Castaway HD")
	text, _ := windows.UTF16PtrFromString("Johnny Castaway HD has no settings yet.")
	windows.MessageBox(windows.HWND(parent), text, title, windows.MB_OK|windows.MB_ICONINFORMATION)
}
