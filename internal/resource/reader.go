package resource

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// reader reads little-endian values from a byte slice. The first error is
// kept and every later read returns zero, so callers check err once.
type reader struct {
	name string // for error messages
	b    []byte
	off  int
	err  error
}

func newReader(name string, b []byte) *reader {
	return &reader{name: name, b: b}
}

func (r *reader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf("%s at offset %d: %s", r.name, r.off, fmt.Sprintf(format, args...))
	}
}

func (r *reader) remaining() int { return len(r.b) - r.off }

func (r *reader) bytes(n int) []byte {
	if r.err != nil {
		return make([]byte, max(n, 0))
	}
	if n < 0 || n > r.remaining() {
		r.fail("need %d bytes, %d left", n, r.remaining())
		return make([]byte, max(n, 0))
	}
	b := r.b[r.off : r.off+n]
	r.off += n
	return b
}

func (r *reader) u8() uint8   { return r.bytes(1)[0] }
func (r *reader) u16() uint16 { return binary.LittleEndian.Uint16(r.bytes(2)) }
func (r *reader) u32() uint32 { return binary.LittleEndian.Uint32(r.bytes(4)) }

// tag reads a 4-byte block tag such as "BMP:" and checks it.
func (r *reader) tag(want string) {
	got := r.bytes(4)
	if r.err == nil && string(got) != want {
		r.off -= 4
		r.fail("expected block %q, found %q", want, got)
	}
}

// cString reads a NUL-terminated string.
func (r *reader) cString() string {
	if r.err != nil {
		return ""
	}
	i := bytes.IndexByte(r.b[r.off:], 0)
	if i < 0 {
		r.fail("unterminated string")
		return ""
	}
	s := string(r.b[r.off : r.off+i])
	r.off += i + 1
	return s
}

// packed reads a compressed block body: u32 size (counting the method byte
// and the u32 that follow), u8 method, u32 decompressed size, then data.
func (r *reader) packed() []byte {
	size := int(r.u32())
	method := r.u8()
	outSize := int(r.u32())
	in := r.bytes(size - 5)
	if r.err != nil {
		return nil
	}
	out, err := decompress(method, in, outSize)
	if err != nil {
		r.fail("%v", err)
	}
	return out
}

// done fails if any bytes are left unread.
func (r *reader) done() error {
	if r.err == nil && r.remaining() != 0 {
		r.fail("%d unexpected bytes at end", r.remaining())
	}
	return r.err
}
