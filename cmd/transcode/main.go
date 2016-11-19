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
	r         io.Reader
	timestamp [4]byte
	n         int
}

func (r *fbsReader) read4() ([]byte, error) {
	b := make([]byte, 4)
	n, err := r.r.Read(b)
	if n < 4 && err == nil {
		err = io.ErrUnexpectedEOF
	}
	return b, err
}

func (r *fbsReader) Read(p []byte) (n int, err error) {
	if r.n == 4 {
		b, err := r.read4()
		if err != nil {
			return 0, err
		}
		copy(r.timestamp[:], b)
		r.n = 0
	}
	if r.n == 0 {
		b, err := r.read4()
		if err != nil {
			return 0, err
		}
		r.n = int(bytes2Uint32(b))
		if r.n == 0 {
			return 0, io.EOF
		}
		r.n += 4 // for the timestamp bytes at the end
	}
	if len(p) > r.n-4 {
		p = p[:r.n-4]
	}
	n, err = r.r.Read(p)
	r.n -= n
	return n, err
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
			if err == io.ErrUnexpectedEOF {
				os.Exit(0)
			}
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

		// ServerInit message
		//	2+2+16+4+N bytes
		b := next(24)
		var pf vncclient.PixelFormat
		check(vncclient.ReadPixelFormat(bytes.NewReader(b[4:20]), &pf))
		if !(pf.BPP == 32 &&
			pf.Depth == 24 &&
			pf.TrueColor &&
			pf.RedMax == 255 &&
			pf.GreenMax == 255 &&
			pf.BlueMax == 255) {
			log.Fatalf("Unsupported pixel format: %#v\n", pf)
		}
		emit(append(b, next(int(bytes2Uint32(b[20:24])))...))
	}

	var fbu vncclient.FramebufferUpdateMessage
	conn := &vncclient.ClientConn{
		Encs: []vncclient.Encoding{
			new(vncclient.TightEncoding),
			new(vncclient.RawEncoding),
			new(cursorEncoding),
		},
		PixelFormat: vncclient.PixelFormat{
			BPP:        32,
			Depth:      24,
			TrueColor:  true,
			RedMax:     255,
			GreenMax:   255,
			BlueMax:    255,
			RedShift:   0,
			GreenShift: 8,
			BlueShift:  16,
		},
	}
	//
	// Most of the processing time is spent here, transcoding FramebufferUpdate messages
	//
	for {
		switch b := nextSafe(1); b[0] {
		// FramebufferUpdate
		case 0:
			msg, err := fbu.Read(conn, r)
			if err == io.EOF {
				return
			}
			check(err)
			rects := msg.(*vncclient.FramebufferUpdateMessage).Rectangles
			size := 4
			for _, r := range rects {
				size += 12
				if ce, ok := r.Enc.(*cursorEncoding); ok {
					size += len(ce.b)
				} else {
					size += r.Area() * 4
				}
			}
			check(binary.Write(w, binary.BigEndian, uint32(size)))
			check(w.Write(b))
			check(w.Write([]byte{0})) // padding
			check(binary.Write(w, binary.BigEndian, uint16(len(rects))))
			for _, r := range rects {
				for _, x := range []interface{}{
					r.X, r.Y, r.Width, r.Height, r.Enc.Type(),
				} {
					check(binary.Write(w, binary.BigEndian, x))
				}
				var colors []vncclient.Color
				switch e := r.Enc.(type) {
				case *vncclient.TightEncoding:
					colors = e.Colors
				case *vncclient.RawEncoding:
					colors = e.Colors
				case *cursorEncoding:
					check(w.Write(e.b))
				}
				for _, c := range colors {
					check(w.Write([]byte{c.R, c.G, c.B, 0}))
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
