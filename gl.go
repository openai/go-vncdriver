// +build !no_gl

package main

import "github.com/openai/go-vncdriver/vncgl"

func (b *sessionInfo) initRenderer(name string) error {
	if b.rendererSet {
		return nil
	}
	b.rendererSet = true

	// Make sure we've initialized glfw
	vncgl.SetupRendering()
	return b.batch.SetRenderer(name, &vncgl.VNCGL{})
}
