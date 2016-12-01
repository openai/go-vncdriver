// +build no_gl

package main

const compiledWithGL = false

func (b *sessionInfo) initRenderer(name string) error {
	return nil
}
