package resource

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// Tag is a numbered label in a script, with its description.
type Tag struct {
	ID   uint16
	Name string
}

// Instr is one decoded script instruction.
type Instr struct {
	Offset int    // byte offset in the script
	Opcode uint16 // full opcode, including the argument-count nibble for TTM
	Args   []uint16
	Str    string // string argument (TTM opcodes ending in 0xF)
	HasStr bool
}

// TTM is an animation script.
type TTM struct {
	Version string
	Pages   uint32
	Tags    []Tag
	Code    []Instr
}

// ADS is a scene script.
type ADS struct {
	Version   string
	Resources []Tag // TTM scripts used by the scene, by slot number
	Tags      []Tag
	Code      []Instr
}

func (r *reader) tags() []Tag {
	r.tag("TAG:")
	r.u32()
	n := int(r.u16())
	tags := make([]Tag, 0, n)
	for range n {
		id := r.u16()
		tags = append(tags, Tag{ID: id, Name: r.cString()})
	}
	return tags
}

func (r *reader) version() string {
	r.tag("VER:")
	n := int(r.u32())
	return cString(r.bytes(n))
}

// DecodeTTM decodes a TTM resource.
//
// Layout: "VER:" u32 n + n bytes, "PAG:" u32 pages + 2 bytes, "TT3:" packed
// code, "TTI:" 4 bytes, "TAG:" u32 size u16 count + count (u16 id, string).
func DecodeTTM(e *Entry) (*TTM, error) {
	r := newReader(e.Name, e.Data)
	t := &TTM{Version: r.version()}
	r.tag("PAG:")
	t.Pages = r.u32()
	r.bytes(2)
	r.tag("TT3:")
	code := r.packed()
	r.tag("TTI:")
	r.bytes(4)
	t.Tags = r.tags()
	if err := r.done(); err != nil {
		return nil, err
	}
	var err error
	if t.Code, err = DisasmTTM(code); err != nil {
		return nil, fmt.Errorf("%s: %w", e.Name, err)
	}
	return t, nil
}

// DisasmTTM splits TTM code into instructions. The low nibble of each u16
// opcode is the number of u16 arguments, or 0xF for one NUL-terminated
// string padded to an even length.
func DisasmTTM(code []byte) ([]Instr, error) {
	var out []Instr
	off := 0
	for off < len(code) {
		if off+2 > len(code) {
			return out, fmt.Errorf("offset %d: truncated opcode", off)
		}
		in := Instr{Offset: off, Opcode: binary.LittleEndian.Uint16(code[off:])}
		off += 2
		if n := int(in.Opcode & 0xF); n == 0xF {
			end := off
			for end < len(code) && code[end] != 0 {
				end++
			}
			if end >= len(code) {
				return out, fmt.Errorf("offset %d: unterminated string", in.Offset)
			}
			in.Str, in.HasStr = string(code[off:end]), true
			used := end + 1 - off
			if used%2 == 1 {
				used++
			}
			if off+used > len(code) {
				return out, fmt.Errorf("offset %d: string padding past end", in.Offset)
			}
			off += used
		} else {
			if off+2*n > len(code) {
				return out, fmt.Errorf("offset %d: opcode 0x%04X needs %d args", in.Offset, in.Opcode, n)
			}
			in.Args = make([]uint16, n)
			for i := range n {
				in.Args[i] = binary.LittleEndian.Uint16(code[off:])
				off += 2
			}
		}
		out = append(out, in)
	}
	return out, nil
}

// DecodeADS decodes an ADS resource.
//
// Layout: "VER:" as in TTM, "ADS:" 4 bytes, "RES:" u32 size u16 count +
// count (u16 id, string), "SCR:" packed code, "TAG:" as in TTM.
func DecodeADS(e *Entry) (*ADS, error) {
	r := newReader(e.Name, e.Data)
	a := &ADS{Version: r.version()}
	r.tag("ADS:")
	r.bytes(4)
	r.tag("RES:")
	r.u32()
	n := int(r.u16())
	for range n {
		id := r.u16()
		a.Resources = append(a.Resources, Tag{ID: id, Name: r.cString()})
	}
	r.tag("SCR:")
	code := r.packed()
	a.Tags = r.tags()
	if err := r.done(); err != nil {
		return nil, err
	}
	var err error
	if a.Code, err = DisasmADS(code); err != nil {
		return nil, fmt.Errorf("%s: %w", e.Name, err)
	}
	return a, nil
}

// ADSArgs gives the number of u16 arguments of each ADS opcode, from
// jc_reborn's ads.c. Any other value in the code is a tag label.
var ADSArgs = map[uint16]int{
	0x1070: 2, 0x1330: 2, 0x1350: 2, 0x1360: 2, 0x1370: 2,
	0x1420: 0, 0x1430: 0, 0x1510: 0, 0x1520: 5,
	0x2005: 4, 0x2010: 3, 0x2014: 0,
	0x3010: 0, 0x3020: 1, 0x30FF: 0,
	0x4000: 3, 0xF010: 0, 0xF200: 1, 0xFFFF: 0, 0xFFF0: 0,
}

// DisasmADS splits ADS code into instructions. Values that aren't opcodes
// are tag labels and become instructions without arguments.
func DisasmADS(code []byte) ([]Instr, error) {
	var out []Instr
	off := 0
	for off < len(code) {
		if off+2 > len(code) {
			return out, fmt.Errorf("offset %d: truncated opcode", off)
		}
		in := Instr{Offset: off, Opcode: binary.LittleEndian.Uint16(code[off:])}
		off += 2
		n := ADSArgs[in.Opcode]
		if off+2*n > len(code) {
			return out, fmt.Errorf("offset %d: opcode 0x%04X needs %d args", in.Offset, in.Opcode, n)
		}
		if n > 0 {
			in.Args = make([]uint16, n)
			for i := range n {
				in.Args[i] = binary.LittleEndian.Uint16(code[off:])
				off += 2
			}
		}
		out = append(out, in)
	}
	return out, nil
}

// IsADSOpcode reports whether v is an ADS opcode rather than a tag label.
func IsADSOpcode(v uint16) bool {
	_, ok := ADSArgs[v]
	return ok
}

// TTMNames and ADSNames name the known opcodes, following jc_reborn.
var TTMNames = map[uint16]string{
	0x0080: "DRAW_BACKGROUND", 0x0110: "PURGE", 0x0FF0: "UPDATE",
	0x1021: "SET_DELAY", 0x1051: "SET_BMP_SLOT", 0x1061: "SET_PALETTE_SLOT",
	0x1101: "LOCAL_TAG", 0x1111: "TAG", 0x1121: "TTM_UNKNOWN_1", 0x1201: "GOTO_TAG",
	0x2002: "SET_COLORS", 0x2012: "SET_FRAME1", 0x2022: "TIMER",
	0x4004: "SET_CLIP_ZONE", 0x4204: "COPY_ZONE_TO_BG", 0x4214: "SAVE_IMAGE1",
	0xA002: "DRAW_PIXEL", 0xA054: "SAVE_ZONE", 0xA064: "RESTORE_ZONE",
	0xA0A4: "DRAW_LINE", 0xA104: "DRAW_RECT", 0xA404: "DRAW_CIRCLE",
	0xA504: "DRAW_SPRITE", 0xA524: "DRAW_SPRITE_FLIP", 0xA601: "CLEAR_SCREEN",
	0xB606: "DRAW_SCREEN", 0xC051: "PLAY_SAMPLE",
	0xF01F: "LOAD_SCREEN", 0xF02F: "LOAD_IMAGE", 0xF05F: "LOAD_PALETTE",
}

var ADSNames = map[uint16]string{
	0x1070: "IF_LASTPLAYED_LOCAL", 0x1330: "IF_UNKNOWN_1", 0x1350: "IF_LASTPLAYED",
	0x1360: "IF_NOT_RUNNING", 0x1370: "IF_IS_RUNNING", 0x1420: "AND", 0x1430: "OR",
	0x1510: "PLAY_SCENE", 0x1520: "ADD_SCENE_LOCAL", 0x2005: "ADD_SCENE",
	0x2010: "STOP_SCENE", 0x2014: "UNKNOWN_2014", 0x3010: "RANDOM_START",
	0x3020: "NOP", 0x30FF: "RANDOM_END", 0x4000: "UNKNOWN_6", 0xF010: "FADE_OUT",
	0xF200: "GOSUB_TAG", 0xFFFF: "END", 0xFFF0: "END_IF",
}

// OpcodeCounts counts how often each opcode occurs.
func OpcodeCounts(code []Instr) map[uint16]int {
	m := map[uint16]int{}
	for _, in := range code {
		m[in.Opcode]++
	}
	return m
}

// SortedOpcodes returns the keys of m in ascending order.
func SortedOpcodes(m map[uint16]int) []uint16 {
	ops := make([]uint16, 0, len(m))
	for op := range m {
		ops = append(ops, op)
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i] < ops[j] })
	return ops
}
