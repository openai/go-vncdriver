package vncclient

import (
	"encoding/binary"
	"io"
	"reflect"
	"time"

	"github.com/juju/errors"
)

// A ServerMessage implements a message sent from the server to the client.
type ServerMessage interface {
	// The type of the message that is sent down on the wire.
	Type() uint8

	// Read reads the contents of the message from the reader. At the point
	// this is called, the message type has already been read from the reader.
	// This should return a new ServerMessage that is the appropriate type.
	Read(*ClientConn, io.Reader) (ServerMessage, error)
}

// FramebufferUpdateMessage consists of a sequence of rectangles of
// pixel data that the client should put into its framebuffer.
type FramebufferUpdateMessage struct {
	Rectangles []Rectangle
}

// Rectangle represents a rectangle of pixel data.
type Rectangle struct {
	X      uint16
	Y      uint16
	Width  uint16
	Height uint16
	Enc    Encoding
}

func (r *Rectangle) Area() int {
	return int(r.Width) * int(r.Height)
}

func (*FramebufferUpdateMessage) Type() uint8 {
	return 0
}

func (*FramebufferUpdateMessage) Read(c *ClientConn, r io.Reader) (ServerMessage, error) {
	start := time.Now().UnixNano()

	// Read off the padding
	var padding [1]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	var numRects uint16
	if err := binary.Read(r, binary.BigEndian, &numRects); err != nil {
		return nil, err
	}
    if numRects > 1000 {
        return nil, errors.Errorf("excessive rectangle count %d", int(numRects));
    }

	// Build the map of encodings supported
	encMap := make(map[int32]Encoding)
	for _, enc := range c.Encs {
		encMap[enc.Type()] = enc
	}

	// We must always support the raw encoding
	rawEnc := new(RawEncoding)
	encMap[rawEnc.Type()] = rawEnc

	// Let's also support ZRLEEnocding
	// zrleEnc := new(ZRLEEncoding)
	// encMap[zrleEnc.Type()] = zrleEnc

	rects := make([]Rectangle, numRects)
	for i := uint16(0); i < numRects; i++ {
		var encodingType int32

		rect := &rects[i]
		data := []interface{}{
			&rect.X,
			&rect.Y,
			&rect.Width,
			&rect.Height,
			&encodingType,
		}

		for _, val := range data {
			if err := binary.Read(r, binary.BigEndian, val); err != nil {
				return nil, err
			}
		}

        // Defend against corrupt rectangles before we try to allocate memory.
        // In the encoding readers we compute int(Width) * int(Height), which will overflow if Width*Height >= (1<<31)
        if int(rect.X) > 5120 || int(rect.Y) > 2880 || int(rect.Width) > 5120 || int(rect.Height) > 2880 {
            return nil, errors.Errorf("excessive rectangle origin %dx%d size %dx%d encoding %v", int(rect.X), int(rect.Y), int(rect.Width), int(rect.Height), encodingType);
        }

		enc, ok := encMap[encodingType]
		if !ok {
			return nil, errors.Errorf("unsupported encoding type: %v", encodingType)
		}

		var err error
		rect.Enc, err = enc.Read(c, rect, r)
		if err != nil {
			return nil, err
		}
	}

	var bytes int
	types := map[string]int{}
	for _, rect := range rects {
		bytes += rect.Enc.Size()
		t := reflect.TypeOf(rect.Enc).String()
		types[t]++
	}

	delta := (time.Now().UnixNano() - start) / 1000 / 1000
	if delta > 1 {
		// log.Infof("Time to parse framebuffer update message: %vms (bytes: %v, rects: %v)", delta, bytes, types)
	}

	return &FramebufferUpdateMessage{rects}, nil
}

// SetColorMapEntriesMessage is sent by the server to set values into
// the color map. This message will automatically update the color map
// for the associated connection, but contains the color change data
// if the consumer wants to read it.
//
// See RFC 6143 Section 7.6.2
type SetColorMapEntriesMessage struct {
	FirstColor uint16
	Colors     []Color
}

func (*SetColorMapEntriesMessage) Type() uint8 {
	return 1
}

func (*SetColorMapEntriesMessage) Read(c *ClientConn, r io.Reader) (ServerMessage, error) {
	// Read off the padding
	var padding [1]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	var result SetColorMapEntriesMessage
	if err := binary.Read(r, binary.BigEndian, &result.FirstColor); err != nil {
		return nil, err
	}

	var numColors uint16
	if err := binary.Read(r, binary.BigEndian, &numColors); err != nil {
		return nil, err
	}

	result.Colors = make([]Color, numColors)
	for i := uint16(0); i < numColors; i++ {

		color := &result.Colors[i]
		data := []interface{}{
			&color.R,
			&color.G,
			&color.B,
		}

		for _, val := range data {
			if err := binary.Read(r, binary.BigEndian, val); err != nil {
				return nil, err
			}
		}

		// Update the connection's color map
		c.ColorMap[result.FirstColor+i] = *color
	}

	return &result, nil
}

// BellMessage signals that an audible bell should be made on the client.
//
// See RFC 6143 Section 7.6.3
type BellMessage byte

func (*BellMessage) Type() uint8 {
	return 2
}

func (*BellMessage) Read(*ClientConn, io.Reader) (ServerMessage, error) {
	return new(BellMessage), nil
}

// ServerCutTextMessage indicates the server has new text in the cut buffer.
//
// See RFC 6143 Section 7.6.4
type ServerCutTextMessage struct {
	Text string
}

func (*ServerCutTextMessage) Type() uint8 {
	return 3
}

func (*ServerCutTextMessage) Read(c *ClientConn, r io.Reader) (ServerMessage, error) {
	// Read off the padding
	var padding [3]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	var textLength uint32
	if err := binary.Read(r, binary.BigEndian, &textLength); err != nil {
		return nil, err
	}

	textBytes := make([]uint8, textLength)
	if err := binary.Read(r, binary.BigEndian, &textBytes); err != nil {
		return nil, err
	}

	return &ServerCutTextMessage{string(textBytes)}, nil
}
