# go-vncdriver

A Go implementation of the Gym VNCDriver Python interface. Should be
fast.

## Installation

By default, `go_vncdriver` will try to include OpenGL rendering. If
that build fails, it will fall back to omitting OpenGL rendering. (You
probably don't care about OpenGL rendering on a server anyway.)

### Dependencies

Make sure you have your Python development headers. On Ubuntu, this is
`sudo apt-get install python-dev`.

If you want to allow OpenGL rendering, installed your X and OpenGL
development headers. On Ubuntu, this is:

```
sudo apt-get install libx11-dev libxcursor-dev libxrandr-dev \
  libxinerama-dev libxi-dev libxxf86vm-dev libgl1-mesa-dev \
  mesa-common-dev
```

If you get an error like the following...
```
VNCSession has no renderer. This likely means your go_vncdriver was installed without the OpenGL viewer. See
https://github.com/openai/universe/tree/master/go-vncdriver for instructions on how to install with the OpenGL viewer.
```
... then the first
thing to try is to reinstall `go-vncdriver` from PyPI. Manually `rm` both `go_vncdriver/` and `go_vncdriver-0.3.2.dist-info/` from your
`site-packages` directory, and then redo `pip install go-vncdriver`. You can find your `site-packages` directory by running something like `python -c "import go_vncdriver; print(go_vncdriver.__file__)"`

## Python versions

`go_vncdriver` currently supports only Python 2.7, but adding Python
3.x compatibility will be easy.
