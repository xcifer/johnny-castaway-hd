package resource

// Decompression for the DGDS resource blocks: method 1 is RLE, method 2 is
// LZW. Both are ported from uncompress.c in jc_reborn by Jeremie Guillaume
// (GPL-3.0), keeping its LZW quirks (the dictionary restarts at 256 after a
// clear code, and the decoder reads one byte ahead). Bounds checks are added.

import (
	"errors"
	"fmt"
)

// Compression methods used in the BIN:, TT3: and SCR: blocks.
const (
	methodNone = 0
	methodRLE  = 1
	methodLZW  = 2
)

var errOverflow = errors.New("decompressed data larger than declared size")

// decompress expands in to exactly outSize bytes using the given method.
func decompress(method uint8, in []byte, outSize int) ([]byte, error) {
	switch method {
	case methodNone:
		if len(in) != outSize {
			return nil, fmt.Errorf("uncompressed block is %d bytes, declared %d", len(in), outSize)
		}
		return append([]byte(nil), in...), nil
	case methodRLE:
		return decompressRLE(in, outSize)
	case methodLZW:
		return decompressLZW(in, outSize)
	}
	return nil, fmt.Errorf("unknown compression method %d", method)
}

// decompressRLE: a control byte with the high bit set repeats the next byte
// (control & 0x7F) times; otherwise the next control bytes are copied.
func decompressRLE(in []byte, outSize int) ([]byte, error) {
	out := make([]byte, 0, outSize)
	i := 0
	for len(out) < outSize {
		if i >= len(in) {
			return nil, fmt.Errorf("rle: input ended after %d of %d bytes", len(out), outSize)
		}
		control := in[i]
		i++
		if control&0x80 != 0 {
			n := int(control & 0x7F)
			if i >= len(in) {
				return nil, fmt.Errorf("rle: input ended inside a run")
			}
			if len(out)+n > outSize {
				return nil, fmt.Errorf("rle: %w", errOverflow)
			}
			b := in[i]
			i++
			for range n {
				out = append(out, b)
			}
		} else {
			n := int(control)
			if i+n > len(in) {
				return nil, fmt.Errorf("rle: input ended inside a literal copy")
			}
			if len(out)+n > outSize {
				return nil, fmt.Errorf("rle: %w", errOverflow)
			}
			out = append(out, in[i:i+n]...)
			i += n
		}
	}
	if i != len(in) {
		return nil, fmt.Errorf("rle: %d input bytes left over", len(in)-i)
	}
	return out, nil
}

const (
	lzwClear    = 256
	lzwMaxCodes = 4096
)

type lzwReader struct {
	in      []byte
	pos     int  // bytes fetched so far, including the look-ahead byte
	current byte // byte being consumed
	nextBit uint
}

func (r *lzwReader) getByte() byte {
	if r.pos >= len(r.in) {
		return 0
	}
	b := r.in[r.pos]
	r.pos++
	return b
}

// getBits reads n bits, least significant first. It fetches the next byte as
// soon as the current one is used up, so pos runs one byte ahead.
func (r *lzwReader) getBits(n uint) uint32 {
	var x uint32
	for i := range n {
		if r.current&(1<<r.nextBit) != 0 {
			x |= 1 << i
		}
		r.nextBit++
		if r.nextBit > 7 {
			r.current = r.getByte()
			r.nextBit = 0
		}
	}
	return x
}

// decompressLZW: variable-width codes (9 to 12 bits), code 256 clears the
// dictionary and skips to the end of the current code group.
func decompressLZW(in []byte, outSize int) ([]byte, error) {
	if outSize == 0 {
		return nil, errors.New("lzw: declared size is 0")
	}
	if len(in) == 0 {
		return nil, errors.New("lzw: no input")
	}

	var table [lzwMaxCodes]struct {
		prefix uint16
		append byte
	}
	stack := make([]byte, 0, lzwMaxCodes)
	out := make([]byte, 0, outSize)

	r := &lzwReader{in: in}
	nBits := uint(9)
	freeEntry := uint32(257)
	bitPos := uint32(0)

	r.current = r.getByte()
	oldCode := uint16(r.getBits(nBits))
	lastByte := oldCode
	out = append(out, byte(oldCode))

	for r.pos < len(in) {
		newCode := uint16(r.getBits(nBits))
		bitPos += uint32(nBits)

		if newCode == lzwClear {
			nBits3 := uint32(nBits) << 3
			skip := nBits3 - (bitPos-1)%nBits3 - 1
			r.getBits(uint(skip))
			nBits = 9
			freeEntry = 256
			bitPos = 0
			continue
		}

		code := newCode
		if uint32(code) >= freeEntry {
			if len(stack) >= lzwMaxCodes {
				break
			}
			stack = append(stack, byte(lastByte))
			code = oldCode
		}
		for code > 255 {
			if code >= lzwMaxCodes || len(stack) >= lzwMaxCodes {
				return nil, fmt.Errorf("lzw: corrupt code %d", code)
			}
			stack = append(stack, table[code].append)
			code = table[code].prefix
		}
		stack = append(stack, byte(code))
		lastByte = code

		for len(stack) > 0 {
			if len(out) >= outSize {
				return out, nil
			}
			out = append(out, stack[len(stack)-1])
			stack = stack[:len(stack)-1]
		}

		if freeEntry < lzwMaxCodes {
			table[freeEntry].prefix = oldCode
			table[freeEntry].append = byte(lastByte)
			freeEntry++
			if freeEntry >= 1<<nBits && nBits < 12 {
				nBits++
				bitPos = 0
			}
		}
		oldCode = newCode
	}

	if len(out) != outSize {
		return nil, fmt.Errorf("lzw: produced %d bytes, declared %d", len(out), outSize)
	}
	return out, nil
}
