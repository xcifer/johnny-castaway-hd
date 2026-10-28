package install

import (
	"bytes"
	"errors"
	"fmt"
)

// Layout of a packed ("$") file from the install floppy. See docs/install-format.md.
var packedMagic = []byte{0x65, 0x5D, 0x13, 0x8C}

const (
	nameLenOffset = 0x1C
	nameOffset    = 0x1D
)

// ErrNotPacked means the data doesn't start with the packed-file signature.
var ErrNotPacked = errors.New("install: not a packed file")

// Packed is one decompressed file from the floppy.
type Packed struct {
	Name     string // file name stored in the header, e.g. "SCRANTIC.EXE"
	Data     []byte
	Trailing int // bytes after the end of the compressed stream
}

// IsPacked reports whether data starts with the packed-file signature.
func IsPacked(data []byte) bool {
	return bytes.HasPrefix(data, packedMagic)
}

// Unpack decompresses a packed file.
func Unpack(data []byte) (*Packed, error) {
	if !IsPacked(data) {
		return nil, ErrNotPacked
	}
	if len(data) <= nameOffset {
		return nil, fmt.Errorf("install: header truncated (%d bytes)", len(data))
	}
	nameLen := int(data[nameLenOffset])
	end := nameOffset + nameLen
	if end >= len(data) {
		return nil, fmt.Errorf("install: header truncated (%d bytes, name needs %d)", len(data), end+1)
	}
	if data[end] != 0 {
		return nil, fmt.Errorf("install: name not NUL-terminated (byte 0x%02X at offset 0x%X)", data[end], end)
	}
	name := string(data[nameOffset:end])

	stream := data[end+1:]
	out, used, err := Explode(stream)
	if err != nil {
		return nil, fmt.Errorf("install: %s: %w", name, err)
	}
	return &Packed{Name: name, Data: out, Trailing: len(stream) - used}, nil
}
