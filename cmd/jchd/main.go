// Command jchd is the Johnny Castaway HD screensaver.
//
// For now: a placeholder window that opens fullscreen and exits on any
// key or mouse click.
package main

import (
	"errors"
	"flag"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Logical resolution of the original game. Game logic always works in this space.
const (
	logicalWidth  = 640
	logicalHeight = 480
)

// Original sky blue, used until real backgrounds are loaded.
var skyBlue = color.RGBA{R: 0x55, G: 0xAA, B: 0xFF, A: 0xFF}

type game struct {
	maxFrames int // exit after this many frames; 0 = run until input
	frames    int
	keys      []ebiten.Key
}

func (g *game) Update() error {
	g.frames++
	if g.maxFrames > 0 && g.frames >= g.maxFrames {
		return ebiten.Termination
	}
	g.keys = inpututil.AppendJustPressedKeys(g.keys[:0])
	if len(g.keys) > 0 {
		return ebiten.Termination
	}
	for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
		if inpututil.IsMouseButtonJustPressed(b) {
			return ebiten.Termination
		}
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(skyBlue)
	ebitenutil.DebugPrintAt(screen, "Johnny Castaway HD\nPress any key to exit.", 8, 8)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return logicalWidth, logicalHeight
}

func main() {
	windowed := flag.Bool("windowed", false, "run in a window instead of fullscreen")
	frames := flag.Int("frames", 0, "exit after this many frames (for automated tests)")
	flag.Parse()

	ebiten.SetWindowTitle("Johnny Castaway HD")
	if *windowed {
		ebiten.SetWindowSize(logicalWidth*2, logicalHeight*2)
	} else {
		ebiten.SetFullscreen(true)
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	}

	g := &game{maxFrames: *frames}
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	log.Printf("exited cleanly after %d frames", g.frames)
}
