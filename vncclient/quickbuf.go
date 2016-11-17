package vncclient

import (
	"io"
	"reflect"
	"unsafe"
)

type QuickBuf struct {
	buf []byte
	off int
}

func (b *QuickBuf) Len() int { return len(b.buf) - b.off }

func (b *QuickBuf) ReadByte() (byte, error) {
	if b.off >= len(b.buf) {
		return 0, io.EOF
	}
	o := b.buf[b.off]

	b.off++
	return o, nil
}

func (b *QuickBuf) ReadColors(n int) ([]Color, error) {
	skip := colorSize * n
	if b.off+skip > len(b.buf) {
		return nil, io.EOF
	}
	ptr := unsafe.Pointer(&b.buf[b.off])

	// raw := b.buf[b.off : b.off+skip]

	// Convert memory into a Color slice without copying
	// (https://github.com/golang/go/issues/13656#issuecomment-165618599)
	h := reflect.SliceHeader{
		Data: uintptr(ptr),
		Len:  n,
		Cap:  n,
	}
	colors := *(*[]Color)(unsafe.Pointer(&h))
	// colors := (*[(1 << 31) / colorSize]Color)(ptr)[:n:n]

	b.off += skip
	return colors, nil
}

func (b *QuickBuf) ReadColor() (Color, error) {
	c := Color{
		R: b.buf[b.off],
		G: b.buf[b.off+1],
		B: b.buf[b.off+2],
	}
	b.off += colorSize
	return c, nil
}

func NewQuickBuf(buf []byte) *QuickBuf { return &QuickBuf{buf: buf} }
