package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewImage(t *testing.T) {
	const dir = "../../gamedata"
	if _, err := os.Stat(filepath.Join(dir, "RESOURCE.MAP")); errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	img, err := previewImage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 640 || b.Dy() != 480 {
		t.Errorf("size %v, want 640x480", b.Size())
	}
	if got, want := fmt.Sprintf("%x", md5.Sum(img.Pix)), "e97f5b8f66f71c5d8301a81b98ca08de"; got != want {
		t.Errorf("pixels md5 %s, want %s", got, want)
	}

	if _, err := previewImage(t.TempDir()); err == nil {
		t.Error("no error without game data")
	}
}
