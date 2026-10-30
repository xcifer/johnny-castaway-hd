package main

import "os"

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
