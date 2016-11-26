# go-vncdriver

A Go implementation of the Gym VNCDriver Python interface. Should be
fast.

## OpenGL viewer

If you get an error of the form:

```VNCSession has no renderer. This likely means your go_vncdriver was installed without the OpenGL viewer. See https://github.com/openai/go-vncdriver for details on how debug```

That means that your `go-vncdriver` was built without OpenGL
support. (The installer will first try to install with OpenGL, but
will fall back to installing without it.)

To figure out what happened, the easiest approach is to clone this
window and run `./build.sh`, which should print out the error upon
installing with OpenGL:

```
git clone git@github.com:openai/go-vncdriver
cd go-vncdriver
./build.sh
```

Once you've fixed the issue, you should reinstall `go-vncdriver` via
`pip install --ignore-installed go-vncdriver`.

## installation

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

## Python versions

`go_vncdriver` currently supports only Python 2.7, but adding Python
3.x compatibility will be easy.
