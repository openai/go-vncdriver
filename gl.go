// +build !no_gl

package main

import "github.com/openai/gym-vnc/go-vncdriver/vncgl"

func (b *sessionInfo) initRenderer(name string) error {
	return b.batch.SetRenderer(name, &vncgl.VNCGL{})
}
