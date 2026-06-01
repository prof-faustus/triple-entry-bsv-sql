// Package cryptocore is the Go side of the shared TE-BSV crypto core.
// Byte-exact contract: spec/ALGORITHMS.md. Cross-impl parity with the TS (and future C) side is
// enforced by the shared vectors in crypto-core/vectors (SYS-TEST-003).
package cryptocore

import (
	"encoding/binary"
	"errors"
	"unicode/utf8"
)

// Writer is an append-only canonical big-endian encoder (ALGORITHMS.md §1).
type Writer struct{ b []byte }

func NewWriter() *Writer { return &Writer{} }

func (w *Writer) U8(x byte) *Writer  { w.b = append(w.b, x); return w }
func (w *Writer) U16(x uint16) *Writer {
	w.b = binary.BigEndian.AppendUint16(w.b, x)
	return w
}
func (w *Writer) U32(x uint32) *Writer {
	w.b = binary.BigEndian.AppendUint32(w.b, x)
	return w
}
func (w *Writer) U64(x uint64) *Writer {
	w.b = binary.BigEndian.AppendUint64(w.b, x)
	return w
}
func (w *Writer) Raw(p []byte) *Writer { w.b = append(w.b, p...); return w }

// Bytes writes a length-prefixed octet string: u32(len) ‖ p.
func (w *Writer) Bytes(p []byte) *Writer { return w.U32(uint32(len(p))).Raw(p) }

// Str writes a length-prefixed UTF-8 string.
func (w *Writer) Str(s string) *Writer { return w.Bytes([]byte(s)) }

func (w *Writer) Finish() []byte { return w.b }

// Reader is a strict canonical decoder; it rejects truncation and (via End) trailing bytes.
type Reader struct {
	buf []byte
	off int
	err error
}

func NewReader(buf []byte) *Reader { return &Reader{buf: buf} }

func (r *Reader) Err() error { return r.err }

func (r *Reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if r.off+n > len(r.buf) {
		r.err = errors.New("decode: truncated")
		return false
	}
	return true
}

func (r *Reader) U8() byte {
	if !r.need(1) {
		return 0
	}
	v := r.buf[r.off]
	r.off++
	return v
}
func (r *Reader) U16() uint16 {
	if !r.need(2) {
		return 0
	}
	v := binary.BigEndian.Uint16(r.buf[r.off:])
	r.off += 2
	return v
}
func (r *Reader) U32() uint32 {
	if !r.need(4) {
		return 0
	}
	v := binary.BigEndian.Uint32(r.buf[r.off:])
	r.off += 4
	return v
}
func (r *Reader) U64() uint64 {
	if !r.need(8) {
		return 0
	}
	v := binary.BigEndian.Uint64(r.buf[r.off:])
	r.off += 8
	return v
}
func (r *Reader) Raw(n int) []byte {
	if !r.need(n) {
		return nil
	}
	out := make([]byte, n)
	copy(out, r.buf[r.off:r.off+n])
	r.off += n
	return out
}
func (r *Reader) Bytes() []byte {
	n := r.U32()
	return r.Raw(int(n))
}
func (r *Reader) Str() string {
	b := r.Bytes()
	if r.err != nil {
		return ""
	}
	if !utf8.Valid(b) {
		r.err = errors.New("decode: invalid utf-8")
		return ""
	}
	return string(b)
}

// End asserts the whole buffer was consumed.
func (r *Reader) End() error {
	if r.err != nil {
		return r.err
	}
	if r.off != len(r.buf) {
		return errors.New("decode: trailing bytes")
	}
	return nil
}
