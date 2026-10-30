//go:build !windows

package main

// ownConsole reports whether the program has a console window to itself,
// which only Windows opens for it.
func ownConsole() bool { return false }
