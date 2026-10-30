//go:build !windows

package main

import "log"

// showSettings shows the settings. Only Windows asks for them.
func showSettings(parent uintptr) {
	log.Print("Johnny Castaway HD has no settings yet.")
}
