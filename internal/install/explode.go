// Package install reads the compressed files on the Screen Antics install floppy.
//
// explode.go decompresses PKWARE Data Compression Library "implode" data. It
// is ported from blast.c by Mark Adler (zlib contrib/blast, version 1.3, zlib
// license). The original's 4 KB output window is replaced by a growing slice;
// behavior is otherwise the same.
package install

import (
	"errors"
	"fmt"
)

const maxBits = 13 // longest Huffman code in any of the fixed tables

var (
	ErrTruncated = errors.New("explode: input ended before the end code")
	errBadCode   = errors.New("explode: invalid Huffman code")
)

// huffman is a canonical Huffman decoding table: count[n] is the number of
// codes of length n, symbol lists the symbols ordered by code.
type huffman struct {
	count  [maxBits + 1]int16
	symbol []int16
}

// construct builds a table from blast's compact representation: each byte
// holds a code length (low nibble) and a repeat count minus one (high nibble).
// It returns the table and how many codes are left unused: 0 for a complete
// code, negative for an over-subscribed one.
func construct(rep []byte) (*huffman, int) {
	var length []int16
	for _, b := range rep {
		for range int(b>>4) + 1 {
			length = append(length, int16(b&15))
		}
	}

	h := &huffman{symbol: make([]int16, len(length))}
	for _, l := range length {
		h.count[l]++
	}
	left := 1
	for l := 1; l <= maxBits; l++ {
		left <<= 1
		left -= int(h.count[l])
		if left < 0 {
			return h, left
		}
	}

	var offs [maxBits + 1]int16
	for l := 1; l < maxBits; l++ {
		offs[l+1] = offs[l] + h.count[l]
	}
	for sym, l := range length {
		if l != 0 {
			h.symbol[offs[l]] = int16(sym)
			offs[l]++
		}
	}
	return h, left
}

var (
	// Code lengths for literals (ASCII mode), lengths and distances.
	litLen = []byte{
		11, 124, 8, 7, 28, 7, 188, 13, 76, 4, 10, 8, 12, 10, 12, 10, 8, 23, 8,
		9, 7, 6, 7, 8, 7, 6, 55, 8, 23, 24, 12, 11, 7, 9, 11, 12, 6, 7, 22, 5,
		7, 24, 6, 11, 9, 6, 7, 22, 7, 11, 38, 7, 9, 8, 25, 11, 8, 11, 9, 12,
		8, 12, 5, 38, 5, 38, 5, 11, 7, 5, 6, 21, 6, 10, 53, 8, 7, 24, 10, 27,
		44, 253, 253, 253, 252, 252, 252, 13, 12, 45, 12, 45, 12, 61, 12, 45,
		44, 173}
	lenLen  = []byte{2, 35, 36, 53, 38, 23}
	distLen = []byte{2, 20, 53, 230, 247, 151, 248}

	// Base value and extra bits for each length symbol.
	lenBase  = [16]int{3, 2, 4, 5, 6, 7, 8, 9, 10, 12, 16, 24, 40, 72, 136, 264}
	lenExtra = [16]uint{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8}

	litCode, _  = construct(litLen)
	lenCode, _  = construct(lenLen)
	distCode, _ = construct(distLen)
)

// endLength is the length value that marks the end of the stream.
const endLength = 519

type bitReader struct {
	in     []byte
	pos    int
	bitbuf uint32
	bitcnt uint
}

// bits returns the next need bits, least significant first.
func (br *bitReader) bits(need uint) (int, error) {
	val := br.bitbuf
	for br.bitcnt < need {
		if br.pos >= len(br.in) {
			return 0, ErrTruncated
		}
		val |= uint32(br.in[br.pos]) << br.bitcnt
		br.pos++
		br.bitcnt += 8
	}
	br.bitbuf = val >> need
	br.bitcnt -= need
	return int(val & (1<<need - 1)), nil
}

// decode reads one symbol. PKWARE stores Huffman codes bit-inverted.
func (br *bitReader) decode(h *huffman) (int, error) {
	code, first, index := 0, 0, 0
	for l := 1; l <= maxBits; l++ {
		b, err := br.bits(1)
		if err != nil {
			return 0, err
		}
		code |= b ^ 1
		count := int(h.count[l])
		if code < first+count {
			return int(h.symbol[index+code-first]), nil
		}
		index += count
		first += count
		first <<= 1
		code <<= 1
	}
	return 0, errBadCode
}

// Explode decompresses one implode stream from the start of in. It returns
// the output and the number of input bytes the stream used, so callers can
// detect trailing data.
func Explode(in []byte) (out []byte, used int, err error) {
	br := &bitReader{in: in}

	lit, err := br.bits(8)
	if err != nil {
		return nil, 0, err
	}
	if lit > 1 {
		return nil, 0, fmt.Errorf("explode: literal mode %d, want 0 or 1", lit)
	}
	dict, err := br.bits(8)
	if err != nil {
		return nil, 0, err
	}
	if dict < 4 || dict > 6 {
		return nil, 0, fmt.Errorf("explode: dictionary size code %d, want 4-6", dict)
	}

	out = make([]byte, 0, len(in)*2)
	for {
		isMatch, err := br.bits(1)
		if err != nil {
			return out, br.pos, err
		}
		if isMatch == 0 {
			var sym int
			if lit == 1 {
				sym, err = br.decode(litCode)
			} else {
				sym, err = br.bits(8)
			}
			if err != nil {
				return out, br.pos, err
			}
			out = append(out, byte(sym))
			continue
		}

		sym, err := br.decode(lenCode)
		if err != nil {
			return out, br.pos, err
		}
		extra, err := br.bits(lenExtra[sym])
		if err != nil {
			return out, br.pos, err
		}
		length := lenBase[sym] + extra
		if length == endLength {
			return out, br.pos, nil
		}

		shift := uint(dict)
		if length == 2 {
			shift = 2
		}
		hi, err := br.decode(distCode)
		if err != nil {
			return out, br.pos, err
		}
		lo, err := br.bits(shift)
		if err != nil {
			return out, br.pos, err
		}
		dist := hi<<shift + lo + 1
		if dist > len(out) {
			return out, br.pos, fmt.Errorf("explode: distance %d reaches before start of output (%d bytes)", dist, len(out))
		}
		// Byte by byte: source and destination may overlap.
		for range length {
			out = append(out, out[len(out)-dist])
		}
	}
}
