//go:build !windows

package main

import (
	"log"
	"os"
	"syscall"
	"time"
)

// showSettings shows the settings. Only Windows asks for them.
func showSettings(parent uintptr) {
	log.Print("Johnny Castaway HD has no settings yet.")
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
