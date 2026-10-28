package resource

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Sound is a WAV file embedded in SCRANTIC.SCR.
type Sound struct {
	ID   int    // resource number in the executable (1-23)
	Data []byte // the resource as stored: a RIFF WAVE file, padded to 512 bytes
}

// soundType is the custom NE resource type number the sounds are stored under.
const soundType = 28677

// Sounds returns the WAV resources of SCRANTIC.SCR, a 16-bit Windows (NE)
// executable, in resource table order.
func Sounds(exe []byte) ([]Sound, error) {
	le := binary.LittleEndian
	if len(exe) < 0x40 || !bytes.HasPrefix(exe, []byte("MZ")) {
		return nil, fmt.Errorf("not an MZ executable")
	}
	ne := int(le.Uint16(exe[0x3C:]))
	if ne+0x40 > len(exe) || string(exe[ne:ne+2]) != "NE" {
		return nil, fmt.Errorf("not an NE executable")
	}
	r := newReader("SCRANTIC.SCR", exe)
	r.off = ne + int(le.Uint16(exe[ne+0x24:]))
	shift := r.u16()
	if shift > 16 {
		return nil, fmt.Errorf("resource alignment shift %d", shift)
	}

	var sounds []Sound
	for {
		typeID := r.u16()
		if r.err != nil {
			return nil, r.err
		}
		if typeID == 0 {
			break
		}
		count := int(r.u16())
		r.u32()
		for range count {
			off := int(r.u16()) << shift
			size := int(r.u16()) << shift
			r.u16()
			id := r.u16()
			r.u32()
			if r.err != nil {
				return nil, r.err
			}
			if typeID != 0x8000|soundType {
				continue
			}
			if off+size > len(exe) {
				return nil, fmt.Errorf("sound %d at %d+%d is outside the file", id&0x7FFF, off, size)
			}
			data := exe[off : off+size]
			if len(data) < 12 || !bytes.HasPrefix(data, []byte("RIFF")) || string(data[8:12]) != "WAVE" {
				return nil, fmt.Errorf("sound %d is not a RIFF WAVE", id&0x7FFF)
			}
			sounds = append(sounds, Sound{ID: int(id & 0x7FFF), Data: data})
		}
	}
	if len(sounds) == 0 {
		return nil, fmt.Errorf("no sound resources found")
	}
	return sounds, nil
}
