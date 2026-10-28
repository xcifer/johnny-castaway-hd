package resource

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/decoded.md5 from the game data")

func TestRLE(t *testing.T) {
	out, err := decompressRLE([]byte{0x83, 0xAA, 0x02, 0x01, 0x02}, 5)
	if err != nil || !bytes.Equal(out, []byte{0xAA, 0xAA, 0xAA, 0x01, 0x02}) {
		t.Errorf("got % X, %v", out, err)
	}
	for _, tc := range []struct {
		name string
		in   []byte
		size int
	}{
		{"truncated run", []byte{0x83}, 3},
		{"truncated copy", []byte{0x03, 0x01}, 3},
		{"overflow", []byte{0x85, 0xAA}, 3},
		{"leftover input", []byte{0x82, 0xAA, 0x00}, 2},
	} {
		if _, err := decompressRLE(tc.in, tc.size); err == nil {
			t.Errorf("%s: want error", tc.name)
		}
	}
}

// packCodes packs 9-bit LZW codes least significant bit first.
func packCodes(codes ...uint16) []byte {
	var out []byte
	var acc uint32
	n := 0
	for _, c := range codes {
		acc |= uint32(c) << n
		n += 9
		for n >= 8 {
			out = append(out, byte(acc))
			acc >>= 8
			n -= 8
		}
	}
	if n > 0 {
		out = append(out, byte(acc))
	}
	return out
}

func TestLZW(t *testing.T) {
	// A, B, then code 257 (= "AB") twice.
	in := packCodes('A', 'B', 257, 257)
	out, err := decompressLZW(in, 6)
	if err != nil || string(out) != "ABABAB" {
		t.Errorf("got %q, %v; want ABABAB", out, err)
	}
	if out, err := decompressLZW(in, 4); err != nil || string(out) != "ABAB" {
		t.Errorf("short output: got %q, %v; want ABAB", out, err)
	}
	if _, err := decompressLZW(in, 7); err == nil {
		t.Error("declared size too large: want error")
	}
	if _, err := decompressLZW(nil, 1); err == nil {
		t.Error("empty input: want error")
	}
}

func TestDecompressMethods(t *testing.T) {
	if out, err := decompress(methodNone, []byte{1, 2}, 2); err != nil || !bytes.Equal(out, []byte{1, 2}) {
		t.Errorf("method 0: got % X, %v", out, err)
	}
	if _, err := decompress(methodNone, []byte{1, 2}, 3); err == nil {
		t.Error("method 0 size mismatch: want error")
	}
	if _, err := decompress(9, []byte{1}, 1); err == nil {
		t.Error("unknown method: want error")
	}
}

func le16(vs ...uint16) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.LittleEndian.AppendUint16(b, v)
	}
	return b
}

func TestDisasmTTM(t *testing.T) {
	var code []byte
	code = append(code, le16(0x1021, 5)...) // SET_DELAY 5
	code = append(code, le16(0xF02F)...)    // LOAD_IMAGE "AB": 3 bytes + 1 pad
	code = append(code, 'A', 'B', 0, 0xEE)
	code = append(code, le16(0xF01F)...) // LOAD_SCREEN "ABC": 4 bytes, no pad
	code = append(code, 'A', 'B', 'C', 0)
	code = append(code, le16(0xA504, 1, 0xFFFE, 3, 4)...) // DRAW_SPRITE with a negative arg
	code = append(code, le16(0x0FF0)...)

	got, err := DisasmTTM(code)
	if err != nil {
		t.Fatal(err)
	}
	want := []Instr{
		{Offset: 0, Opcode: 0x1021, Args: []uint16{5}},
		{Offset: 4, Opcode: 0xF02F, Str: "AB", HasStr: true},
		{Offset: 10, Opcode: 0xF01F, Str: "ABC", HasStr: true},
		{Offset: 16, Opcode: 0xA504, Args: []uint16{1, 0xFFFE, 3, 4}},
		{Offset: 26, Opcode: 0x0FF0, Args: []uint16{}},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}

	for name, bad := range map[string][]byte{
		"missing args":        le16(0x1021),
		"unterminated string": append(le16(0xF02F), 'A', 'B'),
		"padding past end":    append(le16(0xF02F), 'A', 'B', 0),
		"odd length":          {0x10},
	} {
		if _, err := DisasmTTM(bad); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestDisasmADS(t *testing.T) {
	code := le16(3, 0x2005, 1, 2, 3, 4, 0x1510, 0xFFFF)
	got, err := DisasmADS(code)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].Opcode != 3 || IsADSOpcode(got[0].Opcode) ||
		fmt.Sprint(got[1].Args) != "[1 2 3 4]" || got[3].Opcode != 0xFFFF {
		t.Errorf("got %v", got)
	}
	if _, err := DisasmADS(le16(0x2005, 1)); err == nil {
		t.Error("missing args: want error")
	}
}

func TestUnpack4(t *testing.T) {
	// Two 3x1 images share the middle byte.
	data := []byte{0x12, 0x34, 0x56}
	a, next, err := unpack4(data, 0, 3, 1)
	if err != nil || fmt.Sprint(a.Pix) != "[1 2 3]" || next != 3 {
		t.Fatalf("first: %v %d %v", a.Pix, next, err)
	}
	b, next, err := unpack4(data, next, 3, 1)
	if err != nil || fmt.Sprint(b.Pix) != "[4 5 6]" || next != 6 {
		t.Fatalf("second: %v %d %v", b.Pix, next, err)
	}
	if _, _, err := unpack4(data, next, 1, 1); err == nil {
		t.Error("past end: want error")
	}
}

func name13(s string) []byte {
	b := make([]byte, nameLen)
	copy(b, s)
	return b
}

func TestParseArchive(t *testing.T) {
	var res []byte
	addEntry := func(name string, body []byte) uint32 {
		off := uint32(len(res))
		res = append(res, name13(name)...)
		res = binary.LittleEndian.AppendUint32(res, uint32(len(body)))
		res = append(res, body...)
		return off
	}
	off1 := addEntry("A.TTM", []byte{1, 2, 3})
	off2 := addEntry("B.BMP", []byte{4})

	m := []byte{0, 1, 5, 7, 1, 0}
	m = append(m, name13("RESOURCE.001")...)
	m = append(m, le16(2)...)
	for _, off := range []uint32{off1, off2} {
		m = binary.LittleEndian.AppendUint32(m, 0)
		m = binary.LittleEndian.AppendUint32(m, off)
	}

	a, err := Parse(m, res)
	if err != nil {
		t.Fatal(err)
	}
	if a.FileName != "RESOURCE.001" || len(a.Entries) != 2 ||
		a.Entries[0].Name != "A.TTM" || !bytes.Equal(a.Entries[0].Data, []byte{1, 2, 3}) ||
		a.Find("b.bmp") == nil || a.Find("b.bmp").Type() != "BMP" || a.Find("C.SCR") != nil {
		t.Errorf("got %+v", a)
	}

	if _, err := Parse(m, res[:len(res)-1]); err == nil {
		t.Error("truncated resource file: want error")
	}
	if _, err := Parse(append(m, 0), res); err == nil {
		t.Error("extra map bytes: want error")
	}
}

func TestReaderTag(t *testing.T) {
	r := newReader("x", []byte("BMP:"))
	r.tag("SCR:")
	if r.err == nil || !strings.Contains(r.err.Error(), `"SCR:"`) {
		t.Errorf("err = %v", r.err)
	}
}

// Hashes of the sounds in jc_reborn's README, by resource ID in SCRANTIC.SCR.
var jcRebornSounds = map[int]string{
	1: "35d08fdf2b29fc784cbec78b1fe9a7f2", 2: "f93710cc6f70633393423a8a152a2c85",
	3: "05a08cd60579e3ebcf26d650a185df25", 4: "be4dff1a2a8e0fc612993280df721e0d",
	5: "24deaef44c8b5bb84678978564818103", 6: "eb1055b6cf3d6d7361e9a00e8b088036",
	7: "cab94bace3ef401238daded2e2acec34", 8: "39515446ceb703084d446bd3c64bfbb0",
	9: "f86d5ce3a43cbe56a8af996427d5c173", 10: "5b8535f625094aa491bf8e6246342c77",
	11: "8c173a95da644082e573a0a67ee6d6a3", 12: "e064634cfb9125889ce06314ca01a1ea",
	13: "b3db873332dda51e925533c009352c90", 14: "2eabfe83958db0cad77a3a9492d65fe7",
	15: "2497d51f0e1da6b000dae82090531008", 16: "994a5d06f9ff416215f1874bc330e769",
	17: "5e9cb5a08f39cf555c9662d921a0fed7", 18: "80e7eb0e0c384a51e642e982446fcf1d",
	19: "1a3ab0c7cec89d7d1cd620abdd161d91", 20: "a0f4179f4877cf49122cd87ac7908a1e",
	21: "52fc04e523af3b28c4c6758cdbcafb84", 22: "5a6696cda2a07969522ac62db3e66757",
}

const gameData = "../../gamedata"

func loadGameData(t *testing.T) *Archive {
	t.Helper()
	a, err := Load(gameData)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func md5hex(b []byte) string {
	s := md5.Sum(b)
	return hex.EncodeToString(s[:])
}

// decodedHash fingerprints the decoded content of an entry.
func decodedHash(t *testing.T, e *Entry) string {
	var b bytes.Buffer
	switch e.Type() {
	case "BMP":
		s, err := DecodeBMP(e)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range s.Frames {
			fmt.Fprintf(&b, "%dx%d;", f.Width, f.Height)
			b.Write(f.Pix)
		}
	case "SCR":
		img, err := DecodeSCR(e)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&b, "%dx%d;", img.Width, img.Height)
		b.Write(img.Pix)
	case "PAL":
		p, err := DecodePAL(e)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range p {
			b.Write(c[:])
		}
	case "TTM":
		s, err := DecodeTTM(e)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprint(&b, s.Version, s.Pages, s.Tags, s.Code)
	case "ADS":
		s, err := DecodeADS(e)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprint(&b, s.Version, s.Resources, s.Tags, s.Code)
	default:
		b.Write(e.Data)
	}
	return md5hex(b.Bytes())
}

func TestGameDataDecodes(t *testing.T) {
	a := loadGameData(t)

	counts := map[string]int{}
	for i := range a.Entries {
		counts[a.Entries[i].Type()]++
	}
	if got, want := fmt.Sprint(counts), "map[ADS:10 BMP:117 PAL:1 SCR:10 TTM:41 VIN:1]"; got != want {
		t.Errorf("resource counts %s, want %s", got, want)
	}

	for i := range a.Entries {
		e := &a.Entries[i]
		switch e.Type() {
		case "TTM":
			s, err := DecodeTTM(e)
			if err != nil {
				t.Error(err)
				continue
			}
			for _, in := range s.Code {
				if _, ok := TTMNames[in.Opcode]; !ok {
					t.Errorf("%s offset %d: unknown opcode 0x%04X", e.Name, in.Offset, in.Opcode)
				}
				if in.Opcode == 0x1111 && !hasTag(s.Tags, in.Args[0]) {
					t.Errorf("%s offset %d: TAG %d not in tag list", e.Name, in.Offset, in.Args[0])
				}
				if in.HasStr && a.Find(in.Str) == nil && !knownMissing[e.Name+" "+in.Str] {
					t.Errorf("%s offset %d: %q not in archive", e.Name, in.Offset, in.Str)
				}
			}
		case "ADS":
			s, err := DecodeADS(e)
			if err != nil {
				t.Error(err)
				continue
			}
			for _, r := range s.Resources {
				if a.Find(r.Name) == nil {
					t.Errorf("%s: uses %q, not in archive", e.Name, r.Name)
				}
			}
			for _, in := range s.Code {
				if !IsADSOpcode(in.Opcode) && !hasTag(s.Tags, in.Opcode) {
					t.Errorf("%s offset %d: label %d not in tag list", e.Name, in.Offset, in.Opcode)
				}
			}
		}
	}
}

// knownMissing lists references to resources that aren't in the archive.
// FIRE.TTM is a leftover script that no ADS uses (MJFIRE.TTM is the one
// played), so these sheets are never loaded.
var knownMissing = map[string]bool{
	"FIRE.TTM FLAME.BMP":  true,
	"FIRE.TTM FLURRY.BMP": true,
}

// TTMs that no ADS references.
const unusedTTMs = "[FIRE.TTM GJVIS5W.TTM MJCOCO1.TTM]"

func TestGameDataUnusedScripts(t *testing.T) {
	a := loadGameData(t)
	used := map[string]bool{}
	for i := range a.Entries {
		if e := &a.Entries[i]; e.Type() == "ADS" {
			s, err := DecodeADS(e)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range s.Resources {
				used[strings.ToUpper(r.Name)] = true
			}
		}
	}
	var unused []string
	for i := range a.Entries {
		if e := &a.Entries[i]; e.Type() == "TTM" && !used[e.Name] {
			unused = append(unused, e.Name)
		}
	}
	sort.Strings(unused)
	if got := fmt.Sprint(unused); got != unusedTTMs {
		t.Errorf("unused TTMs %s, want %s", got, unusedTTMs)
	}
}

func hasTag(tags []Tag, id uint16) bool {
	for _, t := range tags {
		if t.ID == id {
			return true
		}
	}
	return false
}

// TestGameDataHashes guards against regressions: the decoded content of
// every resource must match testdata/decoded.md5. Run with -update to
// regenerate it after a deliberate change.
func TestGameDataHashes(t *testing.T) {
	a := loadGameData(t)
	var lines []string
	for i := range a.Entries {
		e := &a.Entries[i]
		lines = append(lines, e.Name+" "+decodedHash(t, e))
	}
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"

	golden := filepath.Join("testdata", "decoded.md5")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		wantLines := map[string]bool{}
		for _, l := range strings.Split(string(want), "\n") {
			wantLines[strings.TrimSpace(l)] = true
		}
		for _, l := range lines {
			if !wantLines[l] {
				t.Errorf("changed or new: %s", l)
			}
		}
		t.Error("decoded output differs from testdata/decoded.md5")
	}
}

func TestGameDataSounds(t *testing.T) {
	exe, err := os.ReadFile(filepath.Join(gameData, "SCRANTIC.SCR"))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	if err != nil {
		t.Fatal(err)
	}
	sounds, err := Sounds(exe)
	if err != nil {
		t.Fatal(err)
	}
	if len(sounds) != 23 {
		t.Errorf("%d sounds, want 23", len(sounds))
	}
	for i, s := range sounds {
		if s.ID != i+1 {
			t.Errorf("sound %d has ID %d", i, s.ID)
		}
		want, ok := jcRebornSounds[s.ID]
		if !ok {
			want = "ff792e653b650e709ab0583db5334629" // resource 23, not in jc_reborn's list
		}
		if got := md5hex(s.Data); got != want {
			t.Errorf("sound %d: md5 %s, want %s", s.ID, got, want)
		}
	}
}
