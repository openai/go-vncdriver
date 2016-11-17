package flexzlib

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestInflator(t *testing.T) {
	inflator := NewInflator()

	for i := range dataCompressed {
		compressed, err := base64.StdEncoding.DecodeString(dataCompressed[i])
		if err != nil {
			t.Fatal(err)
		}

		inflated, err := inflator.Inflate(compressed)
		if err != nil {
			t.Fatal(err)
		}

		expectedInflated, err := base64.StdEncoding.DecodeString(dataInflated[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(inflated, expectedInflated) {
			t.Fatalf("Incorrect inflation: actual=%q expected=%q", inflated, expectedInflated)
		}
	}
}
