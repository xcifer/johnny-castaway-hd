// Command jchd is the Johnny Castaway HD screensaver.
//
// It plays the story of Johnny on his island. For checking the original's
// scripts, -ttm plays clips of one animation script and -ads one scene of a
// scene script. Normally any key or mouse click exits. With -debug, keys
// control the display instead (F1 overlay, F2 scaling filter) and Esc exits.
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
	"github.com/xcifer/johnny-castaway-hd/internal/script"
	"github.com/xcifer/johnny-castaway-hd/internal/story"
)

// The game clock runs at 64 ticks a second, and scenes move on once per
// redraw, every script.FrameTicks ticks, as the original does. See
// docs/timing.md.
const ticksPerSecond = script.TicksPerSecond

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
	ttm       string
	ads       string
	left      bool // island moved left, for scripts of scenes that need it
	day       int  // day of the story, instead of the saved one
	date      string
}

// scene is what plays on the screen. Tick advances it by one redraw.
type scene interface {
	Tick()
	Status() string
}

type game struct {
	opt     options
	data    *gameData
	dataErr error
	ttm     *ttmSpec
	ads     *adsSpec
	now     func() time.Time
	day     func() int

	screen   image.Point // screen size in pixels, from LayoutF
	renderer *render.Renderer
	scene    scene
	ticks    int // clock ticks since the scene started
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
	g.ticks = 0
	switch {
	case g.data == nil:
		g.scene = nil
	case g.ttm != nil || g.ads != nil:
		g.scene = newScriptScene(r, g.data, g.opt.seed, g.ttm, g.ads, g.island())
	default:
		g.scene = newStoryScene(r, g.data, rand.New(rand.NewPCG(g.opt.seed, 0)), g.now, g.day)
	}
	return nil
}

// island is where the island is drawn, relative to its usual place.
func (g *game) island() image.Point {
	if g.opt.left {
		return leftIsland
	}
	return image.Point{}
}

func (g *game) Update() error {
	if g.opt.shot != "" {
		done, err := g.screenshot()
		if err != nil {
			return err
		}
		if done {
			return ebiten.Termination
		}
		return nil
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
	g.tick()
	return nil
}

// tick advances the clock, and the scene on every redraw.
func (g *game) tick() {
	if g.scene == nil {
		return
	}
	if g.ticks%script.FrameTicks == 0 {
		g.scene.Tick()
	}
	g.ticks++
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
	ads, ttm, status := "none", "none", ""
	if g.ads != nil {
		ads = g.opt.ads
	}
	if g.ttm != nil {
		ttm = g.opt.ttm
	}
	if g.opt.left {
		ttm += "  island left"
	}
	if g.scene != nil {
		status = g.scene.Status()
	}
	return fmt.Sprintf("FPS %.1f  TPS %.1f  tick %d (%.1f s)\nscreen %dx%d  frame %dx%d  scale %.3g\nfilter %s (F2)\nADS: %s  TTM: %s  %s",
		ebiten.ActualFPS(), ebiten.ActualTPS(), g.ticks, float64(g.ticks)/ticksPerSecond,
		f.Screen.X, f.Screen.Y, w, h, f.Scale, g.renderer.Filter, ads, ttm, status)
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

// shotTicksPerFrame is how many ticks -shot runs in one frame. Drawing is
// only carried out at the end of a frame, so running hours of ticks in one
// would pile up all their drawing in memory.
const shotTicksPerFrame = 1024

// screenshot renders the scene at shotSize after shotTicks ticks, without
// showing it, and saves it as a PNG. It reports whether it is done.
func (g *game) screenshot() (bool, error) {
	if g.data == nil {
		return false, g.dataErr
	}
	if g.renderer == nil {
		if err := g.newScene(g.opt.shotSize.X, g.opt.shotSize.Y); err != nil {
			return false, err
		}
	}
	for n := 0; g.ticks < g.opt.shotTicks && n < shotTicksPerFrame; n++ {
		g.tick()
	}
	if g.ticks < g.opt.shotTicks {
		return false, nil
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
		return false, err
	}
	if err := png.Encode(f, rgba); err != nil {
		f.Close()
		return false, err
	}
	return true, f.Close()
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
	flag.StringVar(&opt.ttm, "ttm", "", "play clips of one animation script over and over: NAME or NAME:TAG,TAG,...")
	flag.StringVar(&opt.ads, "ads", "", "play one scene of a scene script over and over: NAME:SCENE")
	flag.BoolVar(&opt.left, "left", false, "move the island to the left, as scenes such as fishing at C (MJFISHC) need")
	flag.IntVar(&opt.day, "day", 0, "play this day of the story (1-11) instead of the saved one, without saving")
	flag.StringVar(&opt.date, "date", "", `pretend the clock started at this time, such as "2026-12-24 15:30", without saving`)
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
	if opt.day < 0 || opt.day > story.Days {
		log.Fatalf("bad -day %d: the story has days 1 to %d", opt.day, story.Days)
	}
	return opt
}

func main() {
	opt := parseFlags()
	g := &game{opt: opt, overlay: opt.debug, now: time.Now}
	if opt.date != "" {
		start, err := time.ParseInLocation("2006-01-02 15:04", opt.date, time.Local)
		if err != nil {
			log.Fatalf("bad -date %q: use the form 2026-12-24 15:30", opt.date)
		}
		real := time.Now()
		g.now = func() time.Time { return start.Add(time.Since(real)) }
	}
	switch {
	case opt.day > 0:
		g.day = func() int { return opt.day }
	case opt.date != "":
		g.day = func() int { return 1 }
	default:
		g.day = savedDay(g.now)
	}
	g.data, g.dataErr = loadGameData(opt.dataDir)
	if g.dataErr != nil {
		log.Printf("game data: %v", g.dataErr)
	}
	if opt.ttm != "" && opt.ads != "" {
		log.Fatal("-ttm and -ads can't be used together")
	}
	if opt.ttm != "" && g.data != nil {
		spec, err := parseTTMSpec(g.data, opt.ttm)
		if err != nil {
			log.Fatal(err)
		}
		g.ttm = &spec
	}
	if opt.ads != "" && g.data != nil {
		spec, err := parseADSSpec(g.data, opt.ads)
		if err != nil {
			log.Fatal(err)
		}
		g.ads = &spec
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
