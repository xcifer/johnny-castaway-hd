package install

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Test vector from the blast.c documentation.
var blastVector = []byte{0x00, 0x04, 0x82, 0x24, 0x25, 0x8f, 0x80, 0x7f}

func TestExplodeBlastVector(t *testing.T) {
	out, used, err := Explode(blastVector)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), "AIAIAIAIAIAIA"; got != want {
		t.Errorf("output %q, want %q", got, want)
	}
	if used != len(blastVector) {
		t.Errorf("used %d bytes, want %d", used, len(blastVector))
	}
}

func TestExplodeTrailingData(t *testing.T) {
	in := append(append([]byte{}, blastVector...), 0xAA, 0xBB, 0xCC)
	out, used, err := Explode(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "AIAIAIAIAIAIA" || used != len(blastVector) {
		t.Errorf("got %q used=%d, want AIAIAIAIAIAIA used=%d", out, used, len(blastVector))
	}
}

func TestExplodeTruncated(t *testing.T) {
	for n := 0; n < len(blastVector); n++ {
		if _, _, err := Explode(blastVector[:n]); !errors.Is(err, ErrTruncated) {
			t.Errorf("first %d bytes: err = %v, want ErrTruncated", n, err)
		}
	}
}

func TestExplodeBadHeader(t *testing.T) {
	for _, in := range [][]byte{
		{0x02, 0x04, 0x00}, // literal mode 2
		{0x00, 0x03, 0x00}, // dictionary too small
		{0x00, 0x07, 0x00}, // dictionary too large
	} {
		if _, _, err := Explode(in); err == nil || errors.Is(err, ErrTruncated) {
			t.Errorf("% X: err = %v, want a header error", in, err)
		}
	}
}

// The fixed tables must expand to the symbol counts blast.c expects and form
// complete codes; a typo in the transcribed tables would break this.
func TestFixedTables(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rep     []byte
		symbols int
	}{
		{"literal", litLen, 256},
		{"length", lenLen, 16},
		{"distance", distLen, 64},
	} {
		h, left := construct(tc.rep)
		if len(h.symbol) != tc.symbols {
			t.Errorf("%s: %d symbols, want %d", tc.name, len(h.symbol), tc.symbols)
		}
		if left != 0 {
			t.Errorf("%s: code is not complete (left = %d)", tc.name, left)
		}
	}
}

func TestUnpackRejects(t *testing.T) {
	if _, err := Unpack([]byte("MZ not packed")); !errors.Is(err, ErrNotPacked) {
		t.Errorf("non-packed: err = %v, want ErrNotPacked", err)
	}
	short := append(append([]byte{}, packedMagic...), make([]byte, 10)...)
	if _, err := Unpack(short); err == nil {
		t.Error("truncated header: want error")
	}
}

func TestUnpackSynthetic(t *testing.T) {
	header := make([]byte, nameOffset)
	copy(header, packedMagic)
	header[nameLenOffset] = 8
	file := append(header, []byte("TEST.TXT\x00")...)
	file = append(file, blastVector...)

	p, err := Unpack(file)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "TEST.TXT" || string(p.Data) != "AIAIAIAIAIAIA" || p.Trailing != 0 {
		t.Errorf("got name=%q data=%q trailing=%d", p.Name, p.Data, p.Trailing)
	}
}

// Real floppy files are user-supplied and never committed, so this test
// skips when they're absent. The expected hashes come from
// Johnny-Castaway-Enhanced's list of tested files.
func TestUnpackFloppy(t *testing.T) {
	dir := filepath.Join("..", "..", "gamedata", "floppy")
	for _, tc := range []struct{ packed, name, md5 string }{
		{"RESOURCE.00$", "RESOURCE.001", "8bb6c99e9129806b5089a39d24228a36"},
		{"SCRANTIC.SC$", "SCRANTIC.EXE", ""}, // no published hash; checked to be a Windows NE executable
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.packed))
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("%s not present; put the floppy files in gamedata/floppy to run this test", tc.packed)
		}
		if err != nil {
			t.Fatal(err)
		}
		p, err := Unpack(data)
		if err != nil {
			t.Fatalf("%s: %v", tc.packed, err)
		}
		if p.Name != tc.name {
			t.Errorf("%s: name %q, want %q", tc.packed, p.Name, tc.name)
		}
		sum := md5.Sum(p.Data)
		got := hex.EncodeToString(sum[:])
		t.Logf("%s -> %s: %d bytes, md5 %s, %d trailing bytes", tc.packed, p.Name, len(p.Data), got, p.Trailing)
		if tc.md5 != "" && got != tc.md5 {
			t.Errorf("%s: md5 %s, want %s", tc.packed, got, tc.md5)
		}
		if tc.md5 == "" && !isNE(p.Data) {
			t.Errorf("%s: output is not an NE executable", tc.packed)
		}
	}
}

// isNE reports whether data is a 16-bit Windows (NE) executable.
func isNE(data []byte) bool {
	if len(data) < 0x40 || !bytes.HasPrefix(data, []byte("MZ")) {
		return false
	}
	off := int(data[0x3C]) | int(data[0x3D])<<8
	return off+2 <= len(data) && string(data[off:off+2]) == "NE"
}
