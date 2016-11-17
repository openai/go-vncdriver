package flexzlib

import (
	"bytes"
	"io/ioutil"
)

type Inflator struct {
	r *Reader
}

func NewInflator() *Inflator {
	return &Inflator{}
}

func (i *Inflator) Inflate(p []byte) ([]byte, error) {
	buf := bytes.NewBuffer(p)
	if i.r == nil {
		// Need to do this lazily since we need to make sure
		// we have a complete zlib header.
		r, err := NewReader(buf)
		if err != nil {
			return nil, err
		}
		i.r = r
	} else {
		i.r.SwapReader(buf)
	}

	r, e := ioutil.ReadAll(i.r)
	return r, e
}

func (i *Inflator) Read(p []byte) (int, error) {
	return i.r.Read(p)
}
