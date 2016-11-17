package vncclient

import "unsafe"

const colorSize = int(unsafe.Sizeof(Color{}))

// Color represents a single color in a color map.
type Color struct {
	R, G, B uint8
}
