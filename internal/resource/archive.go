// Package resource reads the original game data: the RESOURCE.MAP index, the
// RESOURCE.001 archive and the resources inside it, and the sounds embedded
// in SCRANTIC.SCR. Format notes are in docs/formats.md.
package resource

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	nameLen        = 13 // fixed-size, NUL-padded resource and file names
	entryHeaderLen = nameLen + 4
)

// Entry is one resource in the archive.
type Entry struct {
	Name string // e.g. "JOHNWALK.BMP"
	Data []byte // the resource body, after the name and size header
}

// Type returns the resource type from the name's extension, e.g. "BMP".
func (e *Entry) Type() string {
	if i := strings.LastIndexByte(e.Name, '.'); i >= 0 {
		return e.Name[i+1:]
	}
	return ""
}

// Archive is the parsed contents of RESOURCE.MAP and its resource file.
type Archive struct {
	FileName string // resource file named by the map, e.g. "RESOURCE.001"
	Entries  []Entry
}

// Load reads RESOURCE.MAP and the resource file it names from dir.
func Load(dir string) (*Archive, error) {
	mapData, err := os.ReadFile(filepath.Join(dir, "RESOURCE.MAP"))
	if err != nil {
		return nil, err
	}
	name, err := mapFileName(mapData)
	if err != nil {
		return nil, err
	}
	resData, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	return Parse(mapData, resData)
}

// mapFileName returns the resource file name stored in the map.
func mapFileName(mapData []byte) (string, error) {
	if len(mapData) < 6+nameLen {
		return "", fmt.Errorf("RESOURCE.MAP: too short (%d bytes)", len(mapData))
	}
	return cString(mapData[6 : 6+nameLen]), nil
}

// Parse reads an archive from the contents of the map and resource files.
//
// Map layout: 6 header bytes (byte 4 is the number of resource files; this
// game has one), the 13-byte resource file name, a u16 entry count, then per
// entry a u32 (unused here) and the u32 offset of the entry in the resource
// file. Each entry there starts with its 13-byte name and u32 body size.
func Parse(mapData, resData []byte) (*Archive, error) {
	r := newReader("RESOURCE.MAP", mapData)
	header := r.bytes(6)
	fileName := cString(r.bytes(nameLen))
	count := int(r.u16())
	if r.err != nil {
		return nil, r.err
	}
	if header[4] != 1 {
		return nil, fmt.Errorf("RESOURCE.MAP: %d resource files, only 1 is supported", header[4])
	}

	a := &Archive{FileName: fileName, Entries: make([]Entry, 0, count)}
	for i := range count {
		r.u32()
		offset := int(r.u32())
		if r.err != nil {
			return nil, r.err
		}
		if offset < 0 || offset+entryHeaderLen > len(resData) {
			return nil, fmt.Errorf("%s: entry %d header at offset %d is outside the file", fileName, i, offset)
		}
		er := newReader(fileName, resData[offset:])
		name := cString(er.bytes(nameLen))
		size := int(er.u32())
		body := er.bytes(size)
		if er.err != nil {
			return nil, fmt.Errorf("entry %d (%s): %w", i, name, er.err)
		}
		a.Entries = append(a.Entries, Entry{Name: name, Data: body})
	}
	if r.remaining() != 0 {
		return nil, fmt.Errorf("RESOURCE.MAP: %d bytes after the last entry", r.remaining())
	}
	return a, nil
}

// Find returns the entry with the given name (case-insensitive), or nil.
func (a *Archive) Find(name string) *Entry {
	for i := range a.Entries {
		if strings.EqualFold(a.Entries[i].Name, name) {
			return &a.Entries[i]
		}
	}
	return nil
}

// cString returns b up to its first NUL byte.
func cString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}
