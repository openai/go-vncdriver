package vncclient

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/juju/errors"
	logging "github.com/op/go-logging"
	"github.com/pixiv/go-libjpeg/jpeg"
)

var log = logging.MustGetLogger("vncclient")

// An Encoding implements a method for encoding pixel data that is
// sent by the server to the client.
type Encoding interface {
	// The number that uniquely identifies this encoding type.
	Type() int32

	// Read reads the contents of the encoded pixel data from the reader.
	// This should return a new Encoding implementation that contains
	// the proper data.
	Read(*ClientConn, *Rectangle, io.Reader) (Encoding, error)

	Size() int
}

type QualityLevel uint32

func (f QualityLevel) Size() int {
	return 0
}

func (f QualityLevel) Type() int32 {
	return -32 + int32(f)
}

func (f QualityLevel) Read(c *ClientConn, rect *Rectangle, r io.Reader) (Encoding, error) {
	return f, errors.NotImplementedf("quality level is a pseudo-encoding")
}

// Compression level
type CompressLevel uint32

func (l CompressLevel) Size() int {
	return 0
}

func (l CompressLevel) Type() int32 {
	return -256 + int32(l)
}

func (l CompressLevel) Read(c *ClientConn, rect *Rectangle, r io.Reader) (Encoding, error) {
	return l, errors.NotImplementedf("compress level is a pseudo-encoding")
}

type FineQualityLevel uint32

func (f FineQualityLevel) Size() int {
	return 0
}

func (f FineQualityLevel) Type() int32 {
	return -512 + int32(f)
}

func (f FineQualityLevel) Read(c *ClientConn, rect *Rectangle, r io.Reader) (Encoding, error) {
	return f, errors.NotImplementedf("fine quality level is a pseudo-encoding")
}

type SubsampleLevel uint32

func (s SubsampleLevel) Size() int {
	return 0
}

func (s SubsampleLevel) Type() int32 {
	return -768 + int32(s)
}

func (s SubsampleLevel) Read(c *ClientConn, rect *Rectangle, r io.Reader) (Encoding, error) {
	return s, errors.NotImplementedf("fine quality level is a pseudo-encoding")
}

// RawEncoding is raw pixel data sent by the server.
//
// See RFC 6143 Section 7.7.1
type RawEncoding struct {
	Colors []Color
}

func (r *RawEncoding) Size() int {
	return len(r.Colors) * 3
}

func (*RawEncoding) Type() int32 {
	return 0
}

func (*RawEncoding) Read(c *ClientConn, rect *Rectangle, r io.Reader) (Encoding, error) {
	bytesPerPixel := c.PixelFormat.BPP / 8

	// Various housekeeping helpers
	pixelBytes := make([]uint8, bytesPerPixel)
	var byteOrder binary.ByteOrder = binary.LittleEndian
	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}

	// Output buffer
	colors := make([]Color, rect.Area())

	// Read all needed bytes: this improves performance so we
	// don't have to do piecemeal unbuffered reads.
	buf := make([]byte, rect.Area()*int(bytesPerPixel))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	r = bytes.NewBuffer(buf)

	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
			if _, err := io.ReadFull(r, pixelBytes); err != nil {
				return nil, err
			}

			var rawPixel uint32
			if c.PixelFormat.BPP == 8 {
				rawPixel = uint32(pixelBytes[0])
			} else if c.PixelFormat.BPP == 16 {
				rawPixel = uint32(byteOrder.Uint16(pixelBytes))
			} else if c.PixelFormat.BPP == 32 {
				rawPixel = byteOrder.Uint32(pixelBytes)
			}

			color := &colors[int(y)*int(rect.Width)+int(x)]
			if c.PixelFormat.TrueColor {
				color.R = uint8((rawPixel >> c.PixelFormat.RedShift) & uint32(c.PixelFormat.RedMax))
				color.G = uint8((rawPixel >> c.PixelFormat.GreenShift) & uint32(c.PixelFormat.GreenMax))
				color.B = uint8((rawPixel >> c.PixelFormat.BlueShift) & uint32(c.PixelFormat.BlueMax))
			} else {
				*color = c.ColorMap[rawPixel]
			}
		}
	}

	return &RawEncoding{colors}, nil
}

// ZRLEEncoding is Zlib run-length encoded pixel data
//
// See RFC 6143 Section 7.7.6
type ZRLEEncoding struct {
	Colors []Color
	size   int32
}

func (*ZRLEEncoding) Type() int32 {
	return 16
}

func (z *ZRLEEncoding) Size() int {
	return int(z.size)
}

func (z *ZRLEEncoding) Read(c *ClientConn, rect *Rectangle, r io.Reader) (Encoding, error) {
	var length int32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// Could maybe get by without the copy
	compressed := make([]uint8, length)
	if err := binary.Read(r, binary.BigEndian, &compressed); err != nil {
		return nil, err
	}

	inflated, err := c.inflator.Inflate(compressed)
	if err != nil {
		return nil, errors.Annotate(err, "could not inflate")
	}

	// It's now safe to start reading other ZRLE messages if desired
	log.Debugf("expanded zlib: %d bytes -> %d bytes", len(compressed), len(inflated))

	// TODO: other format checks here
	if c.PixelFormat.BPP < 24 {
		return nil, errors.Errorf("unsupported bitsPerPixel: %d", c.PixelFormat.BPP)
	}

	// data := base64.StdEncoding.EncodeToString(inflated)
	// log.Infof("payload %v %v %v %v: %v", rect.X, rect.Y, rect.Width, rect.Height, data)

	buf := NewQuickBuf(inflated)
	colors, err := z.parse(rect, buf)
	if err != nil {
		return nil, errors.Annotatef(err, "could not parse ZRLEEncoding colors")
	}

	if buf.Len() != 0 {
		return nil, errors.Errorf("BUG: buffer still had %d unread bytes", buf.Len())
	}

	// buf := bytes.NewBuffer(inflated)
	return &ZRLEEncoding{colors, length}, nil
}

func (z *ZRLEEncoding) parse(rect *Rectangle, r *QuickBuf) ([]Color, error) {
	colors := make([]Color, rect.Area())

	// We pass in a scratch buffer so that parseTile doesn't need
	// to allocate its own. A better implementation would probably
	// write directly into the colors buffer.
	scratch := make([]Color, 64*64)

	for tileY := uint16(0); tileY < rect.Height; tileY += 64 {
		tileHeight := min(64, rect.Height-tileY)
		for tileX := uint16(0); tileX < rect.Width; tileX += 64 {
			tileWidth := min(64, rect.Width-tileX)

			err := z.parseTile(rect, colors, r, tileX, tileY, tileWidth, tileHeight, scratch[:int(tileHeight)*int(tileWidth)])
			if err != nil {
				return nil, err
			}
		}
	}

	return colors, nil
}

func (*ZRLEEncoding) parseTile(rect *Rectangle, colors []Color, r *QuickBuf, tileX, tileY, tileWidth, tileHeight uint16, scratch []Color) error {
	// Each tile begins with a subencoding type byte.  The top bit of this
	// byte is set if the tile has been run-length encoded, clear otherwise.
	// The bottom 7 bits indicate the size of the palette used: zero means
	// no palette, 1 means that the tile is of a single color, and 2 to 127
	// indicate a palette of that size.  The special subencoding values 129
	// and 127 indicate that the palette is to be reused from the last tile
	// that had a palette, with and without RLE, respectively.
	subencoding, err := r.ReadByte()
	if err != nil {
		return errors.Annotate(err, "failed to read subencoding")
	}

	runLengthEncoded := subencoding&128 != 0
	paletteSize := subencoding & 127

	paletteData, err := r.ReadColors(int(paletteSize))
	if err != nil {
		return errors.Annotatef(err, "failed to read palette: runLengthEncoded:%v paletteSize:%v", runLengthEncoded, paletteSize)
	}

	if paletteSize == 0 && !runLengthEncoded {
		// 0: Raw pixel data. width*height pixel values follow (where width and
		// height are the width and height of the tile):
		//
		//  +-----------------------------+--------------+-------------+
		//  | No. of bytes                | Type [Value] | Description |
		//  +-----------------------------+--------------+-------------+
		//  | width*height*BytesPerCPixel | CPIXEL array | pixels      |
		//  +-----------------------------+--------------+-------------+

		colors, err := r.ReadColors(len(scratch))
		if err != nil {
			return errors.Annotate(err, "failed to read raw colors")
		}
		// Don't bother with the scratch buffer
		scratch = colors
	} else if paletteSize == 1 && !runLengthEncoded {
		// 1: A solid tile consisting of a single color.  The pixel value
		// follows:
		//
		// +----------------+--------------+-------------+
		// | No. of bytes   | Type [Value] | Description |
		// +----------------+--------------+-------------+
		// | bytesPerCPixel | CPIXEL       | pixelValue  |
		// +----------------+--------------+-------------+
		pixelValue := paletteData[0]
		fillColor(scratch, pixelValue)
	} else if !runLengthEncoded {
		// 2 to 16:  Packed palette types.  The paletteSize is the value of the
		// subencoding, which is followed by the palette, consisting of
		// paletteSize pixel values.  The packed pixels follow, with each
		// pixel represented as a bit field yielding a zero-based index into
		// the palette.  For paletteSize 2, a 1-bit field is used; for
		// paletteSize 3 or 4, a 2-bit field is used; and for paletteSize
		// from 5 to 16, a 4-bit field is used.  The bit fields are packed
		// into bytes, with the most significant bits representing the
		// leftmost pixel (i.e., big endian).  For tiles not a multiple of 8,
		// 4, or 2 pixels wide (as appropriate), padding bits are used to
		// align each row to an exact number of bytes.

		//   +----------------------------+--------------+--------------+
		//   | No. of bytes               | Type [Value] | Description  |
		//   +----------------------------+--------------+--------------+
		//   | paletteSize*bytesPerCPixel | CPIXEL array | palette      |
		//   | m                          | U8 array     | packedPixels |
		//   +----------------------------+--------------+--------------+

		//  where m is the number of bytes representing the packed pixels.
		//  For paletteSize of 2, this is div(width+7,8)*height; for
		//  paletteSize of 3 or 4, this is div(width+3,4)*height; or for
		//  paletteSize of 5 to 16, this is div(width+1,2)*height.

		var bitsPerPackedPixel uint8
		if paletteSize > 16 {
			// No palette reuse in zrle
			bitsPerPackedPixel = 8
		} else if paletteSize > 4 {
			bitsPerPackedPixel = 4
		} else if paletteSize > 2 {
			bitsPerPackedPixel = 2
		} else {
			bitsPerPackedPixel = 1
		}

		for j := uint16(0); j < tileHeight; j++ {
			// We discard any leftover bits for each new line
			var b uint8
			var nbits uint8

			for i := uint16(0); i < tileWidth; i++ {
				if nbits == 0 {
					b, err = r.ReadByte()
					if err != nil {
						return errors.Annotate(err, "failed to read nbits byte")
					}
					nbits = 8
				}
				nbits -= bitsPerPackedPixel
				paletteIdx := (b >> nbits) & ((1 << bitsPerPackedPixel) - 1) & 127
				pixelValue := paletteData[paletteIdx]
				scratch[j*tileWidth+i] = pixelValue
			}
		}
	} else if runLengthEncoded && paletteSize == 0 {
		// 128:  Plain RLE.  The data consists of a number of runs, repeated
		// until the tile is done.  Runs may continue from the end of one row
		// to the beginning of the next.  Each run is represented by a single
		// pixel value followed by the length of the run.  The length is
		// represented as one or more bytes.  The length is calculated as one
		// more than the sum of all the bytes representing the length.  Any
		// byte value other than 255 indicates the final byte.  So for
		// example, length 1 is represented as [0], 255 as [254], 256 as
		// [255,0], 257 as [255,1], 510 as [255,254], 511 as [255,255,0], and
		// so on.
		//
		// +-------------------------+--------------+-----------------------+
		// | No. of bytes            | Type [Value] | Description           |
		// +-------------------------+--------------+-----------------------+
		// | bytesPerCPixel          | CPIXEL       | pixelValue            |
		// | div(runLength - 1, 255) | U8 array     | 255                   |
		// | 1                       | U8           | (runLength-1) mod 255 |
		// +-------------------------+--------------+-----------------------+

		for pos := 0; pos < len(scratch); {
			pixelValue, err := r.ReadColor()
			if err != nil {
				return err
			}

			count := 1
			for b := uint8(255); b == 255; {
				b, err = r.ReadByte()
				if err != nil {
					return errors.Annotate(err, "failed to read rle byte")
				}
				count += int(b)
			}

			fillColor2(scratch[pos:pos+count], pixelValue)
			pos += count
		}
	} else if runLengthEncoded && paletteSize > 1 {
		// 130 to 255:  Palette RLE.  Followed by the palette, consisting of
		// paletteSize = (subencoding - 128) pixel values:
		//
		//   +----------------------------+--------------+-------------+
		//   | No. of bytes               | Type [Value] | Description |
		//   +----------------------------+--------------+-------------+
		//   | paletteSize*bytesPerCPixel | CPIXEL array | palette     |
		//   +----------------------------+--------------+-------------+
		//
		// Following the palette is, as with plain RLE, a number of runs,
		// repeated until the tile is done.  A run of length one is
		// represented simply by a palette index:
		//
		//         +--------------+--------------+--------------+
		//         | No. of bytes | Type [Value] | Description  |
		//         +--------------+--------------+--------------+
		//         | 1            | U8           | paletteIndex |
		//         +--------------+--------------+--------------+
		//
		// A run of length more than one is represented by a palette index
		// with the top bit set, followed by the length of the run as for
		// plain RLE.
		//
		// +-------------------------+--------------+-----------------------+
		// | No. of bytes            | Type [Value] | Description           |
		// +-------------------------+--------------+-----------------------+
		// | 1                       | U8           | paletteIndex + 128    |
		// | div(runLength - 1, 255) | U8 array     | 255                   |
		// | 1                       | U8           | (runLength-1) mod 255 |
		// +-------------------------+--------------+-----------------------+
		for pos := 0; pos < len(scratch); {
			paletteIdx, err := r.ReadByte()
			if err != nil {
				return errors.Annotate(err, "failed to read palette index")
			}

			count := 1
			if paletteIdx&128 != 0 {
				for b := uint8(255); b == 255; {
					b, err = r.ReadByte()
					if err != nil {
						return errors.Annotate(err, "failed to read byte")
					}
					count += int(b)
				}
			}

			paletteIdx &= 127
			pixelValue := paletteData[paletteIdx]
			fillColor(scratch[pos:pos+count], pixelValue)
			pos += count
		}
	} else {
		return errors.Errorf("Unhandled case: runLengthEncoded=%v paletteSize=%v", runLengthEncoded, paletteSize)
	}

	for j := 0; j < int(tileHeight); j++ {
		off := int(tileY)*int(rect.Width) + int(tileX)
		start := j*int(rect.Width) + off
		copy(colors[start:start+int(tileWidth)], scratch[j*int(tileWidth):])
	}

	return nil
}

func min(x, y uint16) uint16 {
	if x > y {
		return y
	}
	return x
}

func fillColor(dst []Color, pixelValue Color) {
	dst[0] = pixelValue
	for bp := 1; bp < len(dst); {
		copy(dst[bp:], dst[:bp])
		bp *= 2
	}
}

func fillColor2(dst []Color, pixelValue Color) {
	for i := range dst {
		dst[i] = pixelValue
	}
}

type readCloseResetter interface {
	io.ReadCloser
	zlib.Resetter
}

// Superceded by the Fine Quality Level / Compress Level options
type JPEGQuality uint8

func (JPEGQuality) Size() int {
	return 0
}

func (j JPEGQuality) Type() int32 {
	return -32 + int32(j)
}

func (j JPEGQuality) Read(*ClientConn, *Rectangle, io.Reader) (Encoding, error) {
	return j, errors.NotImplementedf("jpeg quality is a pseudo-encoding")
}

// TightEncoding provides efficient compression for pixel data.
//
// Spec:
//     https://github.com/rfbproto/rfbproto/blob/master/rfbproto.rst#tight-encoding
type TightEncoding struct {
	Colors []Color

	streams [4]readCloseResetter
	// streamBufs holds the underlying io.Readers for the zlib streams.
	// We can't share the same underlying io.Reader between the streams
	// because zlib does not always read all available data in a given frame.
	//
	// TODO: could skip the bytes.Buffer and read directly from Readers
	// here. But see zlib note:
	//
	//    If r does not implement io.ByteReader, the decompressor may read
	//    more data than necessary from r.
	//
	// Reading more data than necessary might cause it to encounter an
	// unwanted EOF, while providing an unbuffered ByteReader
	// implementationa might be slower than using bytes.Buffer anyway.
	streamBufs [4]*bytes.Buffer

	// reset is a bitmap that represents which zlib streams need
	// to be reset before their next use.
	reset uint8

	buf *bytes.Buffer

	size int
}

func (*TightEncoding) Type() int32 {
	return 7
}

func (t *TightEncoding) Size() int {
	return t.size
}

func (t *TightEncoding) Read(c *ClientConn, rect *Rectangle, r io.Reader) (Encoding, error) {
	t.size = 0
	if t.buf == nil {
		t.buf = bytes.NewBuffer(nil)
		for i := range t.streamBufs {
			t.streamBufs[i] = bytes.NewBuffer(nil)
		}
	}
	// To reduce implementation complexity, the width of any Tight-encoded
	// rectangle cannot exceed 2048 pixels. If a wider rectangle is
	// desired, it must be split into several rectangles and each one
	// should be encoded separately.
	if rect.Width > 2048 {
		return nil, errors.Errorf("rectangle too wide: %vpx. tight-encoded rectangles cannot be wider than 2048 pixels.", rect.Width)
	}

	// To simplify implementation at the cost of full spec compliance,
	// we only accept the simplest case of pixel format.
	if f := c.PixelFormat; f.BPP != 32 || f.Depth != 24 || !f.TrueColor || f.RedMax != 255 || f.GreenMax != 255 || f.BlueMax != 255 {
		return nil, errors.Errorf("this implementation of Tight encoding does not support this pixel format: %#v", f)
	}

	// The first byte of each Tight-encoded rectangle is a compression-
	// control byte:
	//
	//  +---------------------+--------------+---------------------+
	//  | No. of bytes        | Type [Value] | Description         |
	//  +---------------------+--------------+---------------------+
	//  | 1                   | U8           | compression-control |
	//  +---------------------+--------------+---------------------+
	var compressionControl uint8
	if err := binary.Read(r, binary.BigEndian, &compressionControl); err != nil {
		return nil, err
	}
	t.size++

	// The least significant four bits of the compression-control byte
	// inform the client which zlib compression streams should be reset
	// before decoding the rectangle. Each bit is independent and
	// corresponds to a separate zlib stream that should be reset:
	//
	//  +-----+----------------+
	//  | Bit | Description    |
	//  +-----+----------------+
	//  | 0   | Reset stream 0 |
	//  +-----+----------------+
	//  | 1   | Reset stream 1 |
	//  +-----+----------------+
	//  | 2   | Reset stream 2 |
	//  +-----+----------------+
	//  | 3   | Reset stream 3 |
	//  +-----+----------------+
	t.reset |= compressionControl & 0x0F

	// One of three possible compression methods are supported in the Tight
	// encoding. These are BasicCompression, FillCompression and
	// JpegCompression. If the bit 7 (the most significant bit) of the
	// compression-control byte is 0, then the compression type is
	// BasicCompression.
	if compressionControl>>7 == 0 {
		// In that case, bits 7-4 (the most significant four bits) of
		// compression-control should be interpreted as follows:
		//
		//  +------+--------------+------------------+
		//  | Bits | Binary value | Description      |
		//  +------+--------------+------------------+
		//  | 5-4  | 00           | Use stream 0     |
		//  +------+--------------+------------------+
		//  |      | 01           | Use stream 1     |
		//  +------+--------------+------------------+
		//  |      | 10           | Use stream 2     |
		//  +------+--------------+------------------+
		//  |      | 11           | Use stream 3     |
		//  +------+--------------+------------------+
		//  | 6    | 0            | ---              |
		//  +------+--------------+------------------+
		//  |      | 1            | read-filter-id   |
		//  +------+--------------+------------------+
		//  | 7    | 0            | BasicCompression |
		//  +------+--------------+------------------+
		readFilterID := compressionControl>>6 == 1
		stream := compressionControl >> 4 & 0x03
		log.Debugf("BasicCompression")
		return t.readBasicCompression(c, rect, r, readFilterID, stream)
	}

	// Otherwise, if the bit 7 of compression-control is set to 1, then the
	// compression method is either FillCompression or JpegCompression,
	// depending on other bits of the same byte:
	//
	//  +------+--------------+------------------+
	//  | Bits | Binary value | Description      |
	//  +------+--------------+------------------+
	//  | 7-4  | 1000         | FillCompression  |
	//  +------+--------------+------------------+
	//  |      | 1001         | JpegCompression  |
	//  +------+--------------+------------------+
	//  |      | any other    | invalid          |
	//  +------+--------------+------------------+
	switch compressionControl >> 4 {
	// FillCompression
	case 8:
		log.Debugf("FillCompression")
		// If the compression type is FillCompression, then the only
		// pixel value follows, in TPIXEL format. This value applies to
		// all pixels of the rectangle.
		t.buf.Reset()
		fill, err := t.readTPixels(r, 1)
		if err != nil {
			return nil, err
		}
		colors := make([]Color, rect.Area())
		for i := range colors {
			colors[i] = fill[0]
		}
		return &TightEncoding{Colors: colors, size: t.size}, nil
	// JpegCompression
	case 9:
		log.Debugf("JpegCompression")
		// If the compression type is JpegCompression, the following data
		// stream looks like this:
		//
		//  +--------------+----------+----------------------------------+
		//  | No. of bytes | Type     | Description                      |
		//  +--------------+----------+----------------------------------+
		//  | 1-3          |          | length in compact representation |
		//  +--------------+----------+----------------------------------+
		//  | length       | U8 array | jpeg-data                        |
		//  +--------------+----------+----------------------------------+
		//
		// The jpeg-data is a JFIF stream.
		length, err := t.readCompactLength(byteIOReader{Reader: r})
		if err != nil {
			return nil, err
		}
		buf := io.LimitReader(r, int64(length))
		img, err := jpeg.DecodeIntoRGB(buf, &jpeg.DecoderOptions{})
		if err != nil {
			return nil, errors.Annotate(err, "could not decode jpeg")
		} else if img == nil {
			return nil, errors.New("jpeg decoding returned nil (usually a result of the network being closed)")
		}
		t.size += length

		qbuf := NewQuickBuf(img.Pix)
		colors, err := qbuf.ReadColors(rect.Area())
		if err != nil {
			return nil, err
		}
		return &TightEncoding{Colors: colors, size: t.size}, nil
	default:
		return nil, errors.Errorf("invalid compression control byte: %b", compressionControl)
	}
}

func (t *TightEncoding) readBasicCompression(c *ClientConn, rect *Rectangle, r io.Reader, readFilterID bool, stream uint8) (enc Encoding, e error) {
	var filterID uint8
	if readFilterID {
		// If the compression type is BasicCompression and bit 6 (the
		// read-filter-id bit) of the compression-control byte was set
		// to 1, then the next (second) byte specifies filter-id which
		// tells the decoder what filter type was used by the encoder
		// to pre-process pixel data before the compression.
		if err := binary.Read(r, binary.BigEndian, &filterID); err != nil {
			return nil, err
		}
		t.size++
	} else {
		// If bit 6 of the compression-control byte is set to 0 (no
		// filter-id byte), then the CopyFilter is used.
		filterID = 0
	}

	// The filter-id byte can be one of the following:
	//
	//  +--------------+------+---------+------------------------+
	//  | No. of bytes | Type | [Value] | Description            |
	//  +--------------+------+---------+------------------------+
	//  | 1            | U8   |         | filter-id              |
	//  +--------------+------+---------+------------------------+
	//  |              |      | 0       | CopyFilter (no filter) |
	//  +--------------+------+---------+------------------------+
	//  |              |      | 1       | PaletteFilter          |
	//  +--------------+------+---------+------------------------+
	//  |              |      | 2       | GradientFilter         |
	//  +--------------+------+---------+------------------------+
	log.Debug("stream: ", stream)
	switch filterID {
	// CopyFilter
	case 0:
		log.Debug("CopyFilter")
		// When the CopyFilter is active, raw pixel values in TPIXEL
		// format will be compressed.
		size := rect.Area() * 3
		r, err := t.basicCompressionReader(r, size, stream)
		if err != nil {
			return nil, err
		}
		t.buf.Reset()
		colors, err := t.readTPixels(r, size/3)

		// Copy the colors slice. It uses the same underlying memory as
		// t.buf, but when it is used to update a screen later we might
		// have already modified t.buf while reading a new frame.
		return &TightEncoding{Colors: append([]Color{}, colors...), size: t.size}, err
	// PaletteFilter
	case 1:
		log.Debug("PaletteFilter")
		// The PaletteFilter converts true-color pixel data to indexed
		// colors and a palette which can consist of 2..256 colors.
		//
		// When the PaletteFilter is used, the palette is sent before
		// the pixel data. The palette begins with an unsigned byte
		// which value is the number of colors in the palette minus 1
		// (i.e. 1 means 2 colors, 255 means 256 colors in the palette).
		// Then follows the palette itself which consist of pixel values
		// in TPIXEL format.
		var p uint8
		if err := binary.Read(r, binary.BigEndian, &p); err != nil {
			return nil, err
		}
		paletteSize := int(p) + 1
		t.buf.Reset()
		palette, err := t.readTPixels(r, paletteSize)
		if err != nil {
			return nil, err
		}
		palette = append([]Color{}, palette...)

		// If the number of colors is 2, then each pixel is encoded in
		// 1 bit, otherwise 8 bits are used to encode one pixel. 1-bit
		// encoding is performed such way that the most significant
		// bits correspond to the leftmost pixels, and each row of
		// pixels is aligned to the byte boundary.
		size := rect.Area()
		if paletteSize == 2 {
			size = ((int(rect.Width) + 7) / 8) * int(rect.Height)
		}

		r, err := t.basicCompressionReader(r, size, stream)
		if err != nil {
			return nil, err
		}
		if err = t.readToBuf(r, size); err != nil {
			return nil, err
		}
		buf := t.buf.Bytes()
		colors := make([]Color, rect.Area())
		if paletteSize == 2 {
			offset := uint8(8)
			index := -1
			for i := range colors {
				if offset == 0 || i%int(rect.Width) == 0 {
					offset = 8
					index++
				}
				offset--
				colors[i] = palette[(buf[index]>>offset)&0x01]
			}
		} else {
			for i := range colors {
				if int(buf[i]) >= paletteSize {
					return nil, errors.Errorf("invalid index %d in palette of size %d", buf[i], paletteSize)
				}
				colors[i] = palette[uint8(buf[i])]
			}
		}
		return &TightEncoding{Colors: colors, size: t.size}, nil
	// GradientFilter
	case 2:
		log.Debug("GradientFilter")
		// Note: The GradientFilter may only be used when bits-per-
		// pixel is either 16 or 32.
		if c.PixelFormat.BPP != 16 && c.PixelFormat.BPP != 32 {
			return nil, errors.Errorf("can't use GradientFilter with bitsPerPixel of %v", c.PixelFormat.BPP)
		}

		size := rect.Area() * 3
		r, err := t.basicCompressionReader(r, size, stream)
		if err != nil {
			return nil, err
		}
		t.buf.Reset()
		diffs, err := t.readTPixels(r, size)
		if err != nil {
			return nil, err
		}

		// The GradientFilter pre-processes pixel data with a simple
		// algorithm which converts each color component to a
		// difference between a "predicted" intensity and the actual
		// intensity. Such a technique does not affect uncompressed
		// data size, but helps to compress photo-like images better.
		// Pseudo-code for converting intensities to differences
		// follows:
		//
		//     P[i,j] := V[i-1,j] + V[i,j-1] - V[i-1,j-1];
		//     if (P[i,j] < 0) then P[i,j] := 0;
		//     if (P[i,j] > MAX) then P[i,j] := MAX;
		//     D[i,j] := V[i,j] - P[i,j];
		//
		// Here V[i,j] is the intensity of a color component for a
		// pixel at coordinates (i,j). For pixels outside the current
		// rectangle, V[i,j] is assumed to be zero (which is relevant
		// for P[i,0] and P[0,j]). MAX is the maximum intensity value
		// for a color component.
		colors := make([]Color, size/3)
		cr := colorRect{width: int(rect.Width), colors: colors}
		for i := 0; i < int(rect.Height); i++ {
			for j := 0; j < int(rect.Width); j++ {
				for c := 0; c < 3; c++ {
					p := cr.at(i-1, j, c) + cr.at(i, j-1, c) - cr.at(i-1, j-1, c)
					if p < 0 {
						p = 0
					}
					if p > 255 {
						p = 255
					}
					*component(colors[i], c) = *component(diffs[i], c) + p
				}
			}
		}
		return &TightEncoding{Colors: colors, size: t.size}, nil
	default:
		return nil, errors.Errorf("invalid filter-id byte: %b", filterID)
	}

}

// colorRect simplifies gradient filter computation
type colorRect struct {
	width  int
	colors []Color
}

func (r *colorRect) at(y, x, c int) uint8 {
	if y < 0 || x < 0 {
		return 0
	}
	return *component(r.colors[y*r.width+x], c)
}

func component(c Color, x int) *uint8 {
	switch x {
	case 0:
		return &c.R
	case 1:
		return &c.G
	case 2:
		return &c.B
	}
	panic(fmt.Sprintf("bad component number: %v", component))
}

// readCompressedBytes reads compressed data from r.
// func (t *TightEncoding) readCompressedBytes(r io.Reader, size int, stream uint8) ([]byte, error) {

// basicCompressionReader returns an io.Reader that decompresses data from r.
func (t *TightEncoding) basicCompressionReader(r io.Reader, size int, stream uint8) (out io.Reader, err error) {
	// After the pixel data has been filtered with one of the above three
	// filters, it is compressed using the zlib library. But if the data
	// size after applying the filter but before the compression is less
	// then 12, then the data is sent as is, uncompressed.
	if size < 12 {
		return io.LimitReader(r, int64(size)), nil
	}

	// Four separate zlib streams (0..3) can be used and the
	// decoder should read the actual stream id from the
	// compression-control byte (see [NOTE1]).
	//
	// If the compression is not used, then the pixel data is sent
	// as is, otherwise the data stream looks like this:
	//
	//  +--------------+----------+----------------------------------+
	//  | No. of bytes | Type     | Description                      |
	//  +--------------+----------+----------------------------------+
	//  | 1-3          |          | length in compact representation |
	//  +--------------+----------+----------------------------------+
	//  | length       | U8 array | zlibData                         |
	//  +--------------+----------+----------------------------------+
	length, err := t.readCompactLength(byteIOReader{Reader: r})
	if err != nil {
		return nil, errors.Trace(err)
	}
	t.size += length

	buf := t.streamBufs[stream]
	if t.reset&(1<<stream) != 0 {
		buf.Reset()
	}
	buf.Grow(length)
	if _, err = buf.ReadFrom(io.LimitReader(r, int64(length))); err != nil {
		return nil, errors.Trace(err)
	}

	if t.streams[stream] == nil {
		zr, err := zlib.NewReader(buf)
		if err != nil {
			return nil, errors.Trace(err)
		}
		t.streams[stream] = zr.(readCloseResetter)
	}
	// NOTE1: The decoder must reset the zlib streams before
	// decoding the rectangle, if some of the bits 0, 1, 2 and 3 in
	// the compression-control byte are set to 1. Note that the
	// decoder must reset the indicated zlib streams even if the
	// compression type is FillCompression or JpegCompression.
	if t.reset&(1<<stream) != 0 {
		t.streams[stream].Reset(buf, nil)
		t.reset &^= 1 << stream
	}

	return t.streams[stream], nil
}

// readTPixels reads Colors in TPIXEL format from r. Uses t.buf as buffer space.
//
// NOTE: to simplify implementation, it expects the 3-byte-per-pixel version
// of TPIXEL, which means it is not compatible with all pixel formats.
//
// NOTE: the returned []Color uses the same memory as t.buf, and so will
// no longer be valid once t.buf changes.
func (t *TightEncoding) readTPixels(r io.Reader, n int) ([]Color, error) {
	if t.buf.Len() != 0 {
		panic("unread bytes in t.buf before call to readTPixels")
	}
	if err := t.readToBuf(r, n*3); err != nil {
		return nil, err
	}
	t.size += n * 3
	return (&QuickBuf{buf: t.buf.Next(n * 3)}).ReadColors(n)
}

func (t *TightEncoding) readCompactLength(r io.ByteReader) (int, error) {
	// length is compactly represented in one, two or three bytes,
	// according to the following scheme:
	//
	//  +----------------------------+---------------------------+
	//  | Value                      | Description               |
	//  +----------------------------+---------------------------+
	//  | 0xxxxxxx                   | for values 0..127         |
	//  +----------------------------+---------------------------+
	//  | 1xxxxxxx 0yyyyyyy          | for values 128..16383     |
	//  +----------------------------+---------------------------+
	//  | 1xxxxxxx 1yyyyyyy zzzzzzzz | for values 16384..4194303 |
	//  +----------------------------+---------------------------+
	//
	// Here each character denotes one bit, xxxxxxx are the least
	// significant 7 bits of the value (bits 0-6), yyyyyyy are bits 7-13,
	// and zzzzzzzz are the most significant 8 bits (bits 14-21). For
	// example, decimal value 10000 should be represented as two bytes:
	// binary 10010000 01001110, or hexadecimal 90 4E.

	// Implementation adapted from encoding/binary.ReadUvarint.
	var x uint64
	var s uint
	for i := 0; ; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		t.size++
		if b < 0x80 || i == 2 {
			return int(x | uint64(b)<<s), nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
}

func (t *TightEncoding) readToBuf(r io.Reader, n int) error {
	t.buf.Grow(n)
	_, err := t.buf.ReadFrom(io.LimitReader(r, int64(n)))
	return err
}

// byteIOReader implements both io.ByteReader and io.Reader
type byteIOReader struct {
	io.Reader
	buf [1]byte
}

func (b byteIOReader) ReadByte() (byte, error) {
	_, err := b.Read(b.buf[:])
	return b.buf[0], err
}
