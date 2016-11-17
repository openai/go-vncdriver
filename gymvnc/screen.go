package gymvnc

import "github.com/openai/gym-vnc/go-vncdriver/vncclient"

type Screen struct {
	Data   []vncclient.Color
	Width  uint16
	Height uint16
}

func NewScreen(width, height uint16) *Screen {
	return &Screen{
		Data:   make([]vncclient.Color, uint32(width)*uint32(height)),
		Width:  width,
		Height: height,
	}
}
