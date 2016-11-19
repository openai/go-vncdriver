package gymvnc

import "github.com/openai/go-vncdriver/vncclient"

type VNCEvent interface {
	Execute(c *vncclient.ClientConn) error
}

type KeyEvent struct {
	Keysym uint32
	Down   bool
}

func (k KeyEvent) Execute(c *vncclient.ClientConn) error {
	return c.KeyEvent(k.Keysym, k.Down)
}

type PointerEvent struct {
	Mask vncclient.ButtonMask
	X, Y uint16
}

func (k PointerEvent) Execute(c *vncclient.ClientConn) error {
	return c.PointerEvent(k.Mask, k.X, k.Y)
}
