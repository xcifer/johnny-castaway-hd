// Command jcdump extracts every resource of the original game for inspection:
// sprites and backgrounds as PNG, scripts as text, sounds as WAV, plus a
// manifest.json and an index.html gallery.
//
// Usage: jcdump [-data gamedata] [-out gamedata/dump]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// Colors 0-15 are used; index 0 (magenta) is the sprite transparency key.
const (
	numColors   = 16
	transparent = 0
	sheetWidth  = 640
	sheetGap    = 4
)

func main() {
	dataDir := flag.String("data", "gamedata", "folder with RESOURCE.MAP, RESOURCE.001 and SCRANTIC.SCR")
	outDir := flag.String("out", filepath.Join("gamedata", "dump"), "output folder")
	flag.Parse()

	d := &dumper{out: *outDir}
	if err := d.run(*dataDir); err != nil {
		fmt.Fprintln(os.Stderr, "jcdump:", err)
		os.Exit(1)
	}
}

type dumper struct {
	out      string
	opaque   color.Palette // for backgrounds
	keyed    color.Palette // for sprites: index 0 transparent
	manifest manifest
	warnings []string
	ttmOps   map[uint16]int
	adsOps   map[uint16]int
}

type manifest struct {
	Palette     string       `json:"palette"`
	Sprites     []spriteInfo `json:"sprites"`
	Backgrounds []imageInfo  `json:"backgrounds"`
	Scripts     []string     `json:"scripts"`
	Sounds      []soundInfo  `json:"sounds"`
	Other       []string     `json:"other"`
}

type spriteInfo struct {
	Name   string      `json:"name"`
	Sheet  string      `json:"sheet"`
	Frames []imageInfo `json:"frames"`
}

type imageInfo struct {
	Name   string `json:"name,omitempty"`
	Index  int    `json:"index"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	File   string `json:"file"`
}

type soundInfo struct {
	ID    int    `json:"id"`
	File  string `json:"file"`
	Bytes int    `json:"bytes"`
}

func (d *dumper) warn(format string, args ...any) {
	d.warnings = append(d.warnings, fmt.Sprintf(format, args...))
}

func (d *dumper) run(dataDir string) error {
	a, err := resource.Load(dataDir)
	if err != nil {
		return err
	}
	if err := d.loadPalette(a); err != nil {
		return err
	}
	if err := os.MkdirAll(d.out, 0o755); err != nil {
		return err
	}
	d.ttmOps = map[uint16]int{}
	d.adsOps = map[uint16]int{}

	counts := map[string]int{}
	for i := range a.Entries {
		e := &a.Entries[i]
		counts[e.Type()]++
		var err error
		switch e.Type() {
		case "BMP":
			err = d.sprites(e)
		case "SCR":
			err = d.background(e)
		case "PAL":
			err = d.palette(e)
		case "TTM":
			err = d.ttm(e)
		case "ADS":
			err = d.ads(e)
		default:
			err = d.write(filepath.Join("other", e.Name), e.Data)
			d.manifest.Other = append(d.manifest.Other, "other/"+e.Name)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name, err)
		}
	}

	exe, err := os.ReadFile(filepath.Join(dataDir, "SCRANTIC.SCR"))
	if err != nil {
		return err
	}
	sounds, err := resource.Sounds(exe)
	if err != nil {
		return fmt.Errorf("SCRANTIC.SCR: %w", err)
	}
	for _, s := range sounds {
		file := fmt.Sprintf("wav/sound_%02d.wav", s.ID)
		if err := d.write(file, s.Data); err != nil {
			return err
		}
		d.manifest.Sounds = append(d.manifest.Sounds, soundInfo{ID: s.ID, File: file, Bytes: len(s.Data)})
	}

	if err := d.writeOpcodes(); err != nil {
		return err
	}
	m, err := json.MarshalIndent(d.manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := d.write("manifest.json", m); err != nil {
		return err
	}
	if err := d.writeIndex(); err != nil {
		return err
	}

	types := make([]string, 0, len(counts))
	for t := range counts {
		types = append(types, fmt.Sprintf("%d %s", counts[t], t))
	}
	sort.Strings(types)
	fmt.Printf("Dumped %d resources (%s) and %d sounds to %s\n", len(a.Entries), strings.Join(types, ", "), len(sounds), d.out)
	for _, w := range d.warnings {
		fmt.Println("warning:", w)
	}
	return nil
}

func (d *dumper) loadPalette(a *resource.Archive) error {
	for i := range a.Entries {
		if e := &a.Entries[i]; e.Type() == "PAL" {
			p, err := resource.DecodePAL(e)
			if err != nil {
				return err
			}
			for c := range numColors {
				d.opaque = append(d.opaque, p.Color(c))
			}
			d.keyed = append(color.Palette{}, d.opaque...)
			d.keyed[transparent] = color.NRGBA{}
			d.manifest.Palette = e.Name
			return nil
		}
	}
	return fmt.Errorf("no palette in archive")
}

func (d *dumper) write(rel string, data []byte) error {
	path := filepath.Join(d.out, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (d *dumper) writePNG(rel string, img image.Image) error {
	path := filepath.Join(d.out, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func paletted(img *resource.Image, pal color.Palette) *image.Paletted {
	p := image.NewPaletted(image.Rect(0, 0, img.Width, img.Height), pal)
	copy(p.Pix, img.Pix)
	return p
}

func baseName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func (d *dumper) sprites(e *resource.Entry) error {
	s, err := resource.DecodeBMP(e)
	if err != nil {
		return err
	}
	base := baseName(e.Name)
	info := spriteInfo{Name: e.Name, Sheet: "bmp/" + base + ".png"}

	// Lay frames out left to right, wrapping at sheetWidth.
	type placed struct{ x, y int }
	pos := make([]placed, len(s.Frames))
	x, y, rowH, w := 0, 0, 0, 0
	for i, f := range s.Frames {
		if x > 0 && x+f.Width > sheetWidth {
			x, y, rowH = 0, y+rowH+sheetGap, 0
		}
		pos[i] = placed{x, y}
		x += f.Width + sheetGap
		rowH = max(rowH, f.Height)
		w = max(w, x-sheetGap)
	}
	sheet := image.NewPaletted(image.Rect(0, 0, max(w, 1), max(y+rowH, 1)), d.keyed)

	for i, f := range s.Frames {
		file := fmt.Sprintf("bmp/%s/%03d.png", base, i)
		if err := d.writePNG(file, paletted(&f, d.keyed)); err != nil {
			return err
		}
		info.Frames = append(info.Frames, imageInfo{Index: i, Width: f.Width, Height: f.Height, File: file})
		for row := range f.Height {
			off := sheet.PixOffset(pos[i].x, pos[i].y+row)
			copy(sheet.Pix[off:off+f.Width], f.Pix[row*f.Width:(row+1)*f.Width])
		}
	}
	d.manifest.Sprites = append(d.manifest.Sprites, info)
	return d.writePNG(info.Sheet, sheet)
}

func (d *dumper) background(e *resource.Entry) error {
	img, err := resource.DecodeSCR(e)
	if err != nil {
		return err
	}
	file := "scr/" + baseName(e.Name) + ".png"
	d.manifest.Backgrounds = append(d.manifest.Backgrounds, imageInfo{Name: e.Name, Width: img.Width, Height: img.Height, File: file})
	return d.writePNG(file, paletted(img, d.opaque))
}

func (d *dumper) palette(e *resource.Entry) error {
	p, err := resource.DecodePAL(e)
	if err != nil {
		return err
	}
	const swatch = 32
	img := image.NewPaletted(image.Rect(0, 0, numColors*swatch, swatch), d.opaque)
	for i := range img.Pix {
		img.Pix[i] = uint8(i % (numColors * swatch) / swatch)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s: first %d of 256 colors (6-bit VGA values, then 8-bit RGB)\n", e.Name, numColors)
	for i := range numColors {
		c := p.Color(i)
		fmt.Fprintf(&b, "%2d  %2d %2d %2d   #%02X%02X%02X\n", i, p[i][0], p[i][1], p[i][2], c.R, c.G, c.B)
	}
	base := "pal/" + baseName(e.Name)
	if err := d.write(base+".txt", []byte(b.String())); err != nil {
		return err
	}
	return d.writePNG(base+".png", img)
}

func formatArgs(args []uint16) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprint(int16(a))
	}
	return strings.Join(parts, " ")
}

func tagName(tags []resource.Tag, id uint16) (string, bool) {
	for _, t := range tags {
		if t.ID == id {
			return t.Name, true
		}
	}
	return "", false
}

func (d *dumper) ttm(e *resource.Entry) error {
	t, err := resource.DecodeTTM(e)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "; %s  version %s  pages %d\n; %d tags\n\n", e.Name, t.Version, t.Pages, len(t.Tags))
	for _, in := range t.Code {
		d.ttmOps[in.Opcode]++
		name, ok := resource.TTMNames[in.Opcode]
		if !ok {
			name = fmt.Sprintf("OP_%04X", in.Opcode)
			d.warn("%s offset %d: unknown TTM opcode 0x%04X", e.Name, in.Offset, in.Opcode)
		}
		if in.Opcode == 0x1111 {
			b.WriteString("\n")
		}
		line := fmt.Sprintf("%05d  %s", in.Offset, name)
		if in.HasStr {
			line += fmt.Sprintf(" %q", in.Str)
		} else if len(in.Args) > 0 {
			line += " " + formatArgs(in.Args)
		}
		if in.Opcode == 0x1111 || in.Opcode == 0x1101 {
			if n, ok := tagName(t.Tags, in.Args[0]); ok {
				line = fmt.Sprintf("%-40s ; %s", line, n)
			} else if in.Opcode == 0x1111 {
				d.warn("%s offset %d: TAG %d is not in the tag list", e.Name, in.Offset, in.Args[0])
			}
		}
		b.WriteString(line + "\n")
	}
	file := "ttm/" + baseName(e.Name) + ".txt"
	d.manifest.Scripts = append(d.manifest.Scripts, file)
	return d.write(file, []byte(b.String()))
}

func (d *dumper) ads(e *resource.Entry) error {
	a, err := resource.DecodeADS(e)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "; %s  version %s\n; scripts used (slot: name):\n", e.Name, a.Version)
	for _, r := range a.Resources {
		fmt.Fprintf(&b, ";   %d: %s\n", r.ID, r.Name)
	}
	b.WriteString("\n")
	for _, in := range a.Code {
		if !resource.IsADSOpcode(in.Opcode) {
			n, ok := tagName(a.Tags, in.Opcode)
			if !ok {
				d.warn("%s offset %d: label %d is not in the tag list", e.Name, in.Offset, in.Opcode)
			}
			fmt.Fprintf(&b, "\n%05d  TAG %d  ; %s\n", in.Offset, in.Opcode, n)
			continue
		}
		d.adsOps[in.Opcode]++
		line := fmt.Sprintf("%05d    %s", in.Offset, resource.ADSNames[in.Opcode])
		if len(in.Args) > 0 {
			line += " " + formatArgs(in.Args)
		}
		b.WriteString(line + "\n")
	}
	file := "ads/" + baseName(e.Name) + ".txt"
	d.manifest.Scripts = append(d.manifest.Scripts, file)
	return d.write(file, []byte(b.String()))
}

func (d *dumper) writeOpcodes() error {
	var b strings.Builder
	write := func(title string, ops map[uint16]int, names map[uint16]string) {
		fmt.Fprintf(&b, "%s opcodes used (opcode, count, name)\n", title)
		for _, op := range resource.SortedOpcodes(ops) {
			fmt.Fprintf(&b, "  0x%04X %6d  %s\n", op, ops[op], names[op])
		}
		b.WriteString("\n")
	}
	write("TTM", d.ttmOps, resource.TTMNames)
	write("ADS", d.adsOps, resource.ADSNames)
	return d.write("opcodes.txt", []byte(b.String()))
}

var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Johnny Castaway resources</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 16px; background: #222; color: #eee; }
  h1, h2 { font-weight: 600; }
  nav a { color: #9cf; margin-right: 12px; }
  section { margin-bottom: 48px; }
  figure { margin: 0 0 24px; }
  figcaption { font-size: 14px; color: #aaa; margin-bottom: 4px; }
  img { image-rendering: pixelated; max-width: 100%; }
  .sprite img { width: calc(var(--w) * 2px); background: repeating-conic-gradient(#555 0 25%, #777 0 50%) 0 0 / 16px 16px; }
  .bg img { width: 1280px; }
  audio { vertical-align: middle; }
  ul { columns: 4 200px; }
  a { color: #9cf; }
</style>
</head>
<body>
<h1>Johnny Castaway resources</h1>
<nav><a href="#bg">Backgrounds</a><a href="#sprites">Sprites</a><a href="#sounds">Sounds</a><a href="#scripts">Scripts</a></nav>

<section id="bg"><h2>Backgrounds ({{len .Backgrounds}})</h2>
{{range .Backgrounds}}<figure class="bg"><figcaption>{{.Name}} &middot; {{.Width}}&times;{{.Height}}</figcaption><img src="{{.File}}" alt="{{.Name}}"></figure>
{{end}}</section>

<section id="sprites"><h2>Sprite sheets ({{len .Sprites}}), shown at 2&times;</h2>
<p>Checkerboard = transparent. Palette: <img src="pal/{{.PaletteBase}}.png" alt="palette" style="vertical-align:middle"></p>
{{range .Sprites}}<figure class="sprite"><figcaption>{{.Name}} &middot; {{len .Frames}} frames</figcaption><img src="{{.Sheet}}" alt="{{.Name}}" style="--w: {{.SheetWidth}}"></figure>
{{end}}</section>

<section id="sounds"><h2>Sounds ({{len .Sounds}})</h2>
{{range .Sounds}}<p>Resource {{.ID}} <audio controls preload="none" src="{{.File}}"></audio></p>
{{end}}</section>

<section id="scripts"><h2>Scripts ({{len .Scripts}})</h2>
<ul>{{range .Scripts}}<li><a href="{{.}}">{{.}}</a></li>{{end}}</ul>
<p><a href="opcodes.txt">Opcode usage</a> &middot; <a href="manifest.json">manifest.json</a></p>
</section>
</body>
</html>
`))

func (d *dumper) writeIndex() error {
	type sheet struct {
		spriteInfo
		SheetWidth int
	}
	data := struct {
		manifest
		PaletteBase string
		Sprites     []sheet
	}{manifest: d.manifest, PaletteBase: baseName(d.manifest.Palette)}
	for _, s := range d.manifest.Sprites {
		f, err := os.Open(filepath.Join(d.out, filepath.FromSlash(s.Sheet)))
		if err != nil {
			return err
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			return err
		}
		data.Sprites = append(data.Sprites, sheet{s, cfg.Width})
	}
	var b strings.Builder
	if err := indexTemplate.Execute(&b, data); err != nil {
		return err
	}
	return d.write("index.html", []byte(b.String()))
}
