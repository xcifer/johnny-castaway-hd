package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDataDirs(t *testing.T) {
	exe := filepath.Join("C:", "Games", "Johnny", "JohnnyCastawayHD.scr")
	cwd := filepath.Join("C:", "Windows", "System32")
	want := []string{
		filepath.Join("C:", "Games", "Johnny"),
		filepath.Join("C:", "Games", "Johnny", "gamedata"),
		filepath.Join("C:", "Windows", "System32", "gamedata"),
	}
	if got := dataDirs(exe, cwd); !slices.Equal(got, want) {
		t.Errorf("dataDirs = %q; want %q", got, want)
	}
}

func TestFindDataDir(t *testing.T) {
	root := t.TempDir()
	a, b, c := filepath.Join(root, "a"), filepath.Join(root, "b"), filepath.Join(root, "c")
	for _, d := range []string{a, b, c} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dirs := []string{a, b, c, filepath.Join(root, "missing")}
	if d, ok := findDataDir(dirs); ok {
		t.Errorf("found %q with no data anywhere", d)
	}
	// A folder named RESOURCE.MAP isn't the data.
	if err := os.Mkdir(filepath.Join(a, "RESOURCE.MAP"), 0o755); err != nil {
		t.Fatal(err)
	}
	put := func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "RESOURCE.MAP"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put(c)
	if d, ok := findDataDir(dirs); !ok || d != c {
		t.Errorf("findDataDir = %q, %v; want %q", d, ok, c)
	}
	put(b)
	if d, ok := findDataDir(dirs); !ok || d != b {
		t.Errorf("findDataDir = %q, %v; want %q (the earlier place)", d, ok, b)
	}
}
