// Command transcode converts an FBS file to an equivalent version that only uses raw encoding.
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cheggaaa/pb"
	"github.com/openai/go-vncdriver/vncclient"
)

var (
	in  = flag.String("in", "", "path to server.fbs")
	out = flag.String("out", "", "path to output file")
)

type cursorEncoding struct {
	b []byte
}

func (e *cursorEncoding) Type() int32 {
	return -239
}

func (e *cursorEncoding) Size() int {
	return len(e.b)
}

func (e *cursorEncoding) Read(c *vncclient.ClientConn, rect *vncclient.Rectangle, r io.Reader) (vncclient.Encoding, error) {
	sz := rect.Area()*int(c.PixelFormat.BPP)/8 + (int(rect.Width+7)/8)*int(rect.Height)
	b := make([]byte, sz)
	_, err := io.ReadFull(r, b)
	return &cursorEncoding{b}, err
}

type fbsReader struct {
	buf       bytes.Buffer
	r         io.Reader
	timestamp [4]byte
}

func (r *fbsReader) read4() ([]byte, error) {
	b := make([]byte, 4)
	_, err := io.ReadFull(r.r, b)
	return b, err
}

func (r *fbsReader) Read(p []byte) (n int, err error) {
	if r.buf.Len() == 0 {
		// read length
		b, err := r.read4()
		if err != nil {
			return 0, err
		}
		n := int64(bytes2Uint32(b))

		// read data
		if _, err := io.Copy(&r.buf, io.LimitReader(r.r, n)); err != nil {
			return 0, io.ErrUnexpectedEOF
		}

		// read timestamp
		b, err = r.read4()
		if err != nil {
			return 0, err
		}
		copy(r.timestamp[:], b)
	}

	return r.buf.Read(p)
}

func (r *fbsReader) Timestamp() []byte {
	return r.timestamp[:]
}

func main() {
	flag.Parse()
	if *in == "" || *out == "" {
		flag.Usage()
		fmt.Println("\n--in and --out are both required")
		os.Exit(1)
	}

	// Open input file and start streaming progress to stdout.
	var br *bufio.Reader
	{
		fi, err := os.Stat(*in)
		check(err)
		bar := pb.New64(fi.Size())
		bar.SetMaxWidth(100)
		bar.SetUnits(pb.U_BYTES)
		bar.Start()
		defer bar.FinishPrint("")

		f, err := os.Open(*in)
		check(err)
		defer f.Close()
		br = bufio.NewReader(bar.NewProxyReader(f))
	}

	// Open output file and copy header information to it.
	var w *bufio.Writer
	{
		f, err := os.Create(*out)
		check(err)
		defer f.Close()
		w = bufio.NewWriter(f)
		defer w.Flush()

	}
	{
		version := make([]byte, 12)
		check(io.ReadFull(br, version))
		if s := string(version); s != "FBS 001.002\n" {
			log.Fatal("Unrecognized FBS version: " + s)
		}
		check(w.Write(version))
	}
	{
		b, err := br.ReadBytes('\n')
		check(err)
		check(w.Write(b))
	}

	r := &fbsReader{r: br}

	// Utilities.
	var (
		// next returns the next n bytes of data from the server, and updates
		// timestamp to the FBS timestamp of the last of those bytes.
		// It exits non-zero if there are fewer than n bytes left in the file.
		next = func(n int) []byte {
			b := make([]byte, n)
			check(io.ReadFull(r, b))
			return b
		}

		// nextSafe is like next, but exits zero if there are fewer than n bytes
		// left in the file.
		nextSafe = func(n int) []byte {
			b := make([]byte, n)
			_, err := io.ReadFull(r, b)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				os.Exit(0)
			}
			check(err)
			return b
		}

		// emit writes data in FBS format to w.
		emit = func(data []byte) {
			check(binary.Write(w, binary.BigEndian, uint32(len(data))))
			check(w.Write(data))
			check(w.Write(r.timestamp[:]))
		}
	)

	// Read handshake and init
	{
		// ProtocolVersion message
		//	12 bytes
		version := next(12)
		major, minor, err := vncclient.ParseProtocolVersion(version)
		check(err)
		if major != 3 || minor != 3 {
			log.Fatalf("Unsupported RFB protocol version: %v.%v", major, minor)
		}
		emit(version)

		vncAuth := func() {
			// VNC Authentication challenge (we assume this is the security type chosen)
			//	16 bytes
			emit(next(16))

			// SecurityResult message
			//	4 bytes
			b := next(4)
			if result := bytes2Uint32(b); result != 0 {
				log.Fatalf("Unexpected SecurityResult handshake: %v", result)
			}
			emit(b)
		}

		if minor == 3 {
			b := next(4)
			if sec := bytes2Uint32(b); sec != 2 {
				log.Fatalf("Unsupported security-type: %v", sec)
			}
			emit(b)
			vncAuth()
		}

		if minor == 8 {
			// security-types message
			//	n+1 bytes, where n is the value of the first byte
			n := next(1)
			emit(append(n, next(int(n[0]))...))

			vncAuth()
		}
	}

	// ServerInit message
	//	2+2+16+4+N bytes
	b := next(24)
	var pf vncclient.PixelFormat
	check(vncclient.ReadPixelFormat(bytes.NewReader(b[4:20]), &pf))
	if !(pf.BPP == 32 &&
		pf.Depth == 24 &&
		!pf.BigEndian &&
		pf.TrueColor &&
		pf.RedMax == 255 &&
		pf.GreenMax == 255 &&
		pf.BlueMax == 255 &&
		pf.RedShift == 16 &&
		pf.GreenShift == 8 &&
		pf.BlueShift == 0) {
		log.Fatalf("Unsupported pixel format: %#v\n", pf)
	}
	emit(append(b, next(int(bytes2Uint32(b[20:24])))...))

	var fbu vncclient.FramebufferUpdateMessage
	conn := &vncclient.ClientConn{
		Encs: []vncclient.Encoding{
			new(vncclient.TightEncoding),
			new(vncclient.RawEncoding),
			new(cursorEncoding),
		},
		PixelFormat: pf,
	}
	//
	// Most of the processing time is spent here, transcoding FramebufferUpdate messages
	//
	for {
		switch b := nextSafe(1); b[0] {
		// FramebufferUpdate
		case 0:
			msg, err := fbu.Read(conn, r)

			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			check(err)
			rects := msg.(*vncclient.FramebufferUpdateMessage).Rectangles

			// Start counting how many bytes we're going to write
			size := 0
			size += 4 // account for header (see below)
			for _, r := range rects {
				size += 12 // every Rectangle starts out with four 2-byte fields (X, Y, W, H)
				// and a 4-byte encoding type, and then also contains pixels
				if ce, ok := r.Enc.(*cursorEncoding); ok {
					size += len(ce.b) // if it's a cursor encoding, it already got parsed
				} else {
					size += r.Area() * 4 // if it's pixels, then it's four bytes per pixel
				}
			}

			// Write FramebufferUpdate header
			check(binary.Write(w, binary.BigEndian, uint32(size)))
			check(w.Write(b))                                            // message type
			check(w.Write([]byte{0}))                                    // padding
			check(binary.Write(w, binary.BigEndian, uint16(len(rects)))) // number of rectangles

			// Write FramebufferUpdate rectangles
			for _, r := range rects {
				var colors []vncclient.Color

				isCursor := false
				newEncType := int32(0) // regardless of input type, we're outputting raw (unless it's cursor)

				switch e := r.Enc.(type) {
				case *vncclient.TightEncoding:
					colors = e.Colors
				case *vncclient.RawEncoding:
					colors = e.Colors
				case *cursorEncoding:
					isCursor = true
					newEncType = r.Enc.Type() // keep it as cursor type
				}

				for _, x := range []interface{}{
					r.X, r.Y, r.Width, r.Height, newEncType,
				} {
					check(binary.Write(w, binary.BigEndian, x))
				}

				if isCursor {
					check(w.Write(r.Enc.(*cursorEncoding).b))
				}

				for _, c := range colors {
					colorOrder := (uint32(c.R) << conn.PixelFormat.RedShift) +
						(uint32(c.G) << conn.PixelFormat.GreenShift) +
						(uint32(c.B) << conn.PixelFormat.BlueShift)

					// Write the RGB bytes out in little-endian order,
					// having verified "!pf.BigEndian" above
					check(w.Write([]byte{uint8(colorOrder),
						uint8(colorOrder >> 8),
						uint8(colorOrder >> 16),
						0}))
				}
			}
			check(w.Write(r.timestamp[:]))

		// SetColorMapEntries
		case 1:
			log.Fatal("SetColorMapEntries not supported")
		// Bell
		case 2:
			emit(b)
		// ServerCutText
		case 3:
			b = append(b, nextSafe(7)...)
			b = append(b, nextSafe(int(bytes2Uint32(b[4:8])))...)
			emit(b)
		default:
			log.Fatalf("Unrecognized Server-to-Client message type: %v", b[0])
		}
	}
}

func bytes2Uint32(b []byte) (u uint32) {
	if len(b) != 4 {
		panic(fmt.Sprintf("wrong size for []byte: %v", len(b)))
	}
	check(binary.Read(bytes.NewReader(b), binary.BigEndian, &u))
	return
}

func check(e ...interface{}) {
	err := e[len(e)-1]
	if err != nil {
		log.Fatal(err)
	}
}
