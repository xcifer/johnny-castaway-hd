package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
)

// otherScreenArgs are the options for the copy of the screensaver on
// monitor i: its own story on the same day, quiet so that only the main
// monitor's Johnny is heard, or black if Johnny is wanted on the main
// monitor only. It ends with the process pid.
func otherScreenArgs(i, pid, day int, blank bool) []string {
	args := []string{"-monitor", strconv.Itoa(i), "-follow", strconv.Itoa(pid)}
	if blank {
		return append(args, "-blank")
	}
	return append(args, "-mute", "-day", strconv.Itoa(day))
}

// startOtherScreens starts a copy of the screensaver for each monitor but
// the main one. stop is set when any of them ends, so that all screens go
// together. The copies are returned to be ended with this process.
func startOtherScreens(monitors, day int, blank bool, stop *atomic.Bool) []*exec.Cmd {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("other monitors: %v", err)
		return nil
	}
	var cmds []*exec.Cmd
	for i := 1; i < monitors; i++ {
		cmd := exec.Command(exe, otherScreenArgs(i, os.Getpid(), day, blank)...)
		if err := cmd.Start(); err != nil {
			log.Printf("monitor %d: %v", i, err)
			continue
		}
		go func() {
			cmd.Wait()
			stop.Store(true)
		}()
		cmds = append(cmds, cmd)
	}
	return cmds
}

// follow sets stop when the process pid ends.
func follow(pid int, stop *atomic.Bool) {
	go func() {
		if err := waitProcess(pid); err != nil {
			log.Printf("-follow %d: %v", pid, err)
		}
		stop.Store(true)
	}()
}
