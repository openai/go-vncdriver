package flexzlib

import (
	"bufio"
	"errors"
	"hash"
	"hash/adler32"
	"io"

	"github.com/op/go-logging"
	"github.com/openai/gym-vnc/go-vncdriver/flexflate"
)

var log = logging.MustGetLogger("flexzlib")

const zlibDeflate = 8

var (
	// ErrChecksum is returned when reading ZLIB data that has an invalid checksum.
	ErrChecksum = errors.New("zlib: invalid checksum")
	// ErrDictionary is returned when reading ZLIB data that has an invalid dictionary.
	ErrDictionary = errors.New("zlib: invalid dictionary")
	// ErrHeader is returned when reading ZLIB data that has an invalid header.
	ErrHeader = errors.New("zlib: invalid header")
)

type Reader struct {
	r            flexflate.Reader
	decompressor io.ReadCloser
	digest       hash.Hash32
	err          error
	scratch      [4]byte
}

func (z *Reader) SwapReader(r io.Reader) {
	if fr, ok := r.(flexflate.Reader); ok {
		z.r = fr
	} else {
		z.r = bufio.NewReader(r)
	}

	if z.decompressor != nil {
		z.decompressor.(flexflate.ReaderSwapper).SwapReader(z.r)
	}

	z.err = nil
}

// Resetter resets a ReadCloser returned by NewReader or NewReaderDict to
// to switch to a new underlying Reader. This permits reusing a ReadCloser
// instead of allocating a new one.
type Resetter interface {
	// Reset discards any buffered data and resets the Resetter as if it was
	// newly initialized with the given reader.
	Reset(r io.Reader, dict []byte) error
}

// NewReader creates a new ReadCloser.
// Reads from the returned ReadCloser read and decompress data from r.
// The implementation buffers input and may read more data than necessary from r.
// It is the caller's responsibility to call Close on the ReadCloser when done.
//
// The ReadCloser returned by NewReader also implements Resetter.
func NewReader(r io.Reader) (*Reader, error) {
	return NewReaderDict(r, nil)
}

// NewReaderDict is like NewReader but uses a preset dictionary.
// NewReaderDict ignores the dictionary if the compressed data does not refer to it.
// If the compressed data refers to a different dictionary, NewReaderDict returns ErrDictionary.
//
// The ReadCloser returned by NewReaderDict also implements Resetter.
func NewReaderDict(r io.Reader, dict []byte) (*Reader, error) {
	z := new(Reader)
	err := z.Reset(r, dict)
	if err != nil {
		return nil, err
	}
	return z, nil
}

func (z *Reader) Read(p []byte) (n int, err error) {
	if z.err != nil {
		return 0, z.err
	}
	if len(p) == 0 {
		return 0, nil
	}

	n, err = z.decompressor.Read(p)
	z.digest.Write(p[0:n])

	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}

	if n != 0 || err != io.EOF {
		z.err = err
		return
	}

	// Do not need checksuming
	return

	// Finished file; check checksum.
	if _, err := io.ReadFull(z.r, z.scratch[0:4]); err != nil {
		z.err = err
		return 0, err
	}
	// ZLIB (RFC 1950) is big-endian, unlike GZIP (RFC 1952).
	checksum := uint32(z.scratch[0])<<24 | uint32(z.scratch[1])<<16 | uint32(z.scratch[2])<<8 | uint32(z.scratch[3])
	if checksum != z.digest.Sum32() {
		z.err = ErrChecksum
		return 0, z.err
	}
	return
}

// Calling Close does not close the wrapped io.Reader originally passed to NewReader.
func (z *Reader) Close() error {
	if z.err != nil {
		return z.err
	}
	z.err = z.decompressor.Close()
	return z.err
}

func (z *Reader) Reset(r io.Reader, dict []byte) error {
	z.SwapReader(r)
	_, err := io.ReadFull(z.r, z.scratch[0:2])
	if err != nil {
		return err
	}
	h := uint(z.scratch[0])<<8 | uint(z.scratch[1])
	if (z.scratch[0]&0x0f != zlibDeflate) || (h%31 != 0) {
		return ErrHeader
	}
	haveDict := z.scratch[1]&0x20 != 0
	if haveDict {
		_, err = io.ReadFull(z.r, z.scratch[0:4])
		if err != nil {
			return err
		}
		checksum := uint32(z.scratch[0])<<24 | uint32(z.scratch[1])<<16 | uint32(z.scratch[2])<<8 | uint32(z.scratch[3])
		if checksum != adler32.Checksum(dict) {
			return ErrDictionary
		}
	}
	if z.decompressor == nil {
		if haveDict {
			z.decompressor = flexflate.NewReaderDict(z.r, dict)
		} else {
			z.decompressor = flexflate.NewReader(z.r)
		}
	} else {
		z.decompressor.(flexflate.Resetter).Reset(z.r, dict)
	}
	z.digest = adler32.New()
	return nil
}
