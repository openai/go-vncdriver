package vncclient

import (
	"bytes"
	"encoding/base64"
	"testing"
)

type zrlePayload struct {
	data                string
	x, y, width, height uint16
	result              []Color
}

func TestTightEncodingCompactLength(t *testing.T) {
	cases := []struct {
		in  []byte
		out int
	}{
		{[]byte{0x90, 0x4E}, 10000},
		{[]byte{127}, 127},
	}
	for _, tt := range cases {
		got, err := (&TightEncoding{}).readCompactLength(bytes.NewReader(tt.in))
		if err != nil {
			t.Errorf("&TightEncoding.readCompactLength(%v): unexpected error: %v", tt.in, err)
		} else if got != tt.out {
			t.Errorf("&TightEncoding.readCompactLength(%v) = %v, want %v", tt.in, got, tt.out)
		}
	}
}

func TestZRLEPayload(t *testing.T) {
	for i, payload := range zrlePayloads {
		rect := &Rectangle{
			X:      payload.x,
			Y:      payload.y,
			Width:  payload.width,
			Height: payload.height,
		}

		data, err := base64.StdEncoding.DecodeString(payload.data)
		if err != nil {
			t.Fatal(err)
		}
		buf := &QuickBuf{buf: data}

		encoding := &ZRLEEncoding{}
		parsed, err := encoding.parse(rect, buf)
		if err != nil {
			t.Fatal(err)
		}

		if payload.result == nil {
			t.Fatalf("no result recorded for %d", i)
		}

		for i, expected := range payload.result {
			actual := parsed[i]
			if actual != expected {
				t.Fatalf("mismatch at index %d: expected: %+v actual: %+v", i, expected, actual)
			}
		}
	}
}

var zrlePayloads = []zrlePayload{
	zrlePayload{
		data:   "Bv////j4+Pj59gAAAPn4+Pb5+AAREiERAwIiIREDMCIhEQMzAiERAzMwIREDMzMBFAMzMzAUAzMzMwQDMzMzMAMzMwAAAzAzAREDAQMwEQARAzAREREQMwERERAzARERFQAR",
		x:      285,
		y:      279,
		width:  10,
		height: 16,
		result: []Color{Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 249, B: 246}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}, Color{R: 248, G: 248, B: 248}},
	},
	zrlePayload{
		data:   "A////13qpgAAEQaqqqoKqqqqCqqqqgqqqqoKqqqqCqqqqgqqqqoKqqqqCqqqqgqqqqoKqqqqCqqqqgqqqqoKqqqqCqqqqgqqqqo=",
		x:      712,
		y:      610,
		width:  16,
		height: 16,
		result: []Color{
			Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 255, G: 255, B: 255}, Color{R: 255, G: 255, B: 255}, Color{R: 93, G: 234, B: 166}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17}, Color{R: 0, G: 0, B: 17},
		},
	},
}
