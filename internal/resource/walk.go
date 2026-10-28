package resource

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// WalkStep is one frame of Johnny walking on the island.
type WalkStep struct {
	Flip   bool // draw the sprite mirrored
	Sprite int  // frame in JOHNWALK.BMP
	X, Y   int  // position; X is 0 on the step that ends a path
}

// The walk table is a data block inside SCRANTIC.SCR, not a resource, so its
// location is only known for the version we have checked. Location from
// jc_reborn's extract_walk_data.c; 489 steps of u16 (flip<<15 | sprite),
// u16 x, u16 y.
const (
	walkTableOffset = 0x188EA
	walkTableSteps  = 489
	scranticMD5     = "48b6e36cac54544834c431f6f49c7ada"
)

// WalkTable reads the walking animation table from SCRANTIC.SCR.
func WalkTable(exe []byte) ([]WalkStep, error) {
	sum := md5.Sum(exe)
	if got := hex.EncodeToString(sum[:]); got != scranticMD5 {
		return nil, fmt.Errorf("walk table: unknown SCRANTIC.SCR version (md5 %s)", got)
	}
	steps := make([]WalkStep, walkTableSteps)
	for i := range steps {
		b := exe[walkTableOffset+6*i:]
		w := binary.LittleEndian.Uint16(b)
		steps[i] = WalkStep{
			Flip:   w&0x8000 != 0,
			Sprite: int(w & 0x7FFF),
			X:      int(binary.LittleEndian.Uint16(b[2:])),
			Y:      int(binary.LittleEndian.Uint16(b[4:])),
		}
	}
	return steps, nil
}

// WalkPaths splits the table into paths at the steps whose X is 0.
func WalkPaths(steps []WalkStep) [][]WalkStep {
	var paths [][]WalkStep
	start := 0
	for i, s := range steps {
		if s.X == 0 {
			if i > start {
				paths = append(paths, steps[start:i])
			}
			start = i + 1
		}
	}
	if start < len(steps) {
		paths = append(paths, steps[start:])
	}
	return paths
}
