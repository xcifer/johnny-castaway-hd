// Command jchd is the Johnny Castaway HD screensaver.
//
// For now it shows the island with Johnny walking. Normally any key or mouse
// click exits. With -debug, keys control the display instead (F1 overlay,
// F2 scaling filter) and Esc exits.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/xcifer/johnny-castaway-hd/internal/render"
)

// ticksPerSecond is the game clock: one tick is 20 ms, as in jc_reborn.
// See docs/timing.md.
const ticksPerSecond = 50

type options struct {
	dataDir   string
	windowed  bool
	debug     bool
	frames    int
	filter    render.Filter
	seed      uint64
	shot      string
	shotSize  image.Point
	shotTicks int
}

type game struct {
	opt     options
	data    *gameData
	dataErr error

	screen   image.Point // screen size in pixels, from LayoutF
	renderer *render.Renderer
	scene    *islandScene
	overlay  bool
	frames   int
	start    time.Time // first tick
	keys     []ebiten.Key
}

func (g *game) newScene(w, h int) error {
	r, err := render.New(w, h)
	if err != nil {
		return err
	}
	r.Filter = g.opt.filter
	g.renderer = r
	if g.data != nil {
		g.scene = newIslandScene(r, g.data, rand.New(rand.NewPCG(g.opt.seed, 0)))
	}
	return nil
}

func (g *game) Update() error {
	if g.opt.shot != "" {
		if err := g.screenshot(); err != nil {
			return err
		}
		return ebiten.Termination
	}

	if g.frames == 0 {
		g.start = time.Now()
	}
	g.frames++
	if g.opt.frames > 0 && g.frames >= g.opt.frames {
		return ebiten.Termination
	}
	if g.quitRequested() {
		return ebiten.Termination
	}
	if g.opt.debug {
		if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
			g.overlay = !g.overlay
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) && g.renderer != nil {
			g.renderer.Filter = g.renderer.Filter.Next()
		}
	}

	if g.screen.X == 0 {
		return nil // no layout yet
	}
	if g.renderer == nil || g.renderer.Frame.Screen != g.screen {
		if err := g.newScene(g.screen.X, g.screen.Y); err != nil {
			return err
		}
	}
	if g.scene != nil {
		g.scene.Tick()
	}
	return nil
}

func (g *game) quitRequested() bool {
	if g.opt.debug {
		return inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	}
	g.keys = inpututil.AppendJustPressedKeys(g.keys[:0])
	if len(g.keys) > 0 {
		return true
	}
	for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
		if inpututil.IsMouseButtonJustPressed(b) {
			return true
		}
	}
	return false
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.renderer == nil {
		screen.Fill(color.Black)
		return
	}
	g.renderer.Compose(screen)
	if g.data == nil {
		g.drawText(screen, fmt.Sprintf("Johnny Castaway HD\n\nGame data not loaded:\n%v", g.dataErr))
	}
	if g.overlay {
		g.drawText(screen, g.status())
	}
}

func (g *game) status() string {
	f := g.renderer.Frame
	w, h := f.Size()
	scene := "none"
	tick := 0
	if g.scene != nil {
		scene = g.scene.Status()
		tick = g.scene.tick
	}
	return fmt.Sprintf("FPS %.1f  TPS %.1f  tick %d (%.1f s)\nscreen %dx%d  frame %dx%d  scale %.3g\nfilter %s (F2)\nADS: none  TTM: none  %s",
		ebiten.ActualFPS(), ebiten.ActualTPS(), tick, float64(tick)/ticksPerSecond,
		f.Screen.X, f.Screen.Y, w, h, f.Scale, g.renderer.Filter, scene)
}

// drawText prints text at the top left, scaled up to stay readable.
func (g *game) drawText(screen *ebiten.Image, text string) {
	lines := strings.Split(text, "\n")
	width := 0
	for _, l := range lines {
		width = max(width, len(l))
	}
	img := ebiten.NewImage(width*6+8, len(lines)*16+8)
	defer img.Deallocate()
	img.Fill(color.NRGBA{A: 0xB0})
	ebitenutil.DebugPrintAt(img, text, 4, 4)
	k := max(1, screen.Bounds().Dy()/540)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(k), float64(k))
	op.GeoM.Translate(float64(8*k), float64(8*k))
	screen.DrawImage(img, op)
}

func (g *game) LayoutF(outsideW, outsideH float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	w, h := math.Ceil(outsideW*s), math.Ceil(outsideH*s)
	g.screen = image.Pt(int(w), int(h))
	return w, h
}

func (g *game) Layout(outsideW, outsideH int) (int, int) {
	panic("LayoutF is used instead")
}

// screenshot renders the scene at shotSize after shotTicks ticks, without
// showing it, and saves it as a PNG.
func (g *game) screenshot() error {
	if g.data == nil {
		return g.dataErr
	}
	if err := g.newScene(g.opt.shotSize.X, g.opt.shotSize.Y); err != nil {
		return err
	}
	for range g.opt.shotTicks {
		g.scene.Tick()
	}
	img := ebiten.NewImage(g.opt.shotSize.X, g.opt.shotSize.Y)
	g.renderer.Compose(img)
	if g.overlay {
		g.drawText(img, g.status())
	}
	rgba := image.NewRGBA(img.Bounds())
	img.ReadPixels(rgba.Pix)

	f, err := os.Create(g.opt.shot)
	if err != nil {
		return err
	}
	if err := png.Encode(f, rgba); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func parseFlags() options {
	var opt options
	var filter, shotSize string
	flag.StringVar(&opt.dataDir, "data", "gamedata", "folder with RESOURCE.MAP, RESOURCE.001 and SCRANTIC.SCR")
	flag.BoolVar(&opt.windowed, "windowed", false, "run in a window instead of fullscreen")
	flag.BoolVar(&opt.debug, "debug", false, "keys control the display (F1 overlay, F2 filter, Esc exits)")
	flag.IntVar(&opt.frames, "frames", 0, "exit after this many ticks (for automated tests)")
	flag.StringVar(&filter, "filter", "sharp", "scaling filter: sharp, nearest or smooth")
	flag.Uint64Var(&opt.seed, "seed", 0, "random seed (0 = random)")
	flag.StringVar(&opt.shot, "shot", "", "render one frame to this PNG file and exit")
	flag.StringVar(&shotSize, "shotsize", "1920x1080", "screen size for -shot")
	flag.IntVar(&opt.shotTicks, "shotticks", 0, "ticks to run before -shot")
	flag.Parse()

	switch filter {
	case "sharp":
		opt.filter = render.SharpBilinear
	case "nearest":
		opt.filter = render.Nearest
	case "smooth":
		opt.filter = render.Smooth
	default:
		log.Fatalf("unknown filter %q", filter)
	}
	if _, err := fmt.Sscanf(shotSize, "%dx%d", &opt.shotSize.X, &opt.shotSize.Y); err != nil || opt.shotSize.X <= 0 || opt.shotSize.Y <= 0 {
		log.Fatalf("bad -shotsize %q", shotSize)
	}
	if opt.seed == 0 {
		opt.seed = rand.Uint64()
	}
	return opt
}

func main() {
	opt := parseFlags()
	g := &game{opt: opt, overlay: opt.debug}
	g.data, g.dataErr = loadGameData(opt.dataDir)
	if g.dataErr != nil {
		log.Printf("game data: %v", g.dataErr)
	}

	ebiten.SetTPS(ticksPerSecond)
	ebiten.SetWindowTitle("Johnny Castaway HD")
	switch {
	case opt.shot != "":
		ebiten.SetWindowSize(320, 180)
	case opt.windowed:
		ebiten.SetWindowSize(1280, 720)
	default:
		ebiten.SetFullscreen(true)
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	}

	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if g.frames > 1 {
		secs := time.Since(g.start).Seconds()
		log.Printf("exited cleanly after %d ticks in %.2f s (%.2f ticks/s)", g.frames, secs, float64(g.frames-1)/secs)
	} else {
		log.Printf("exited cleanly")
	}
}
