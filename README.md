# go-vncdriver

A fast VNC driver.

## OpenGL viewer

If you get an error of the form:

```go_vncdriver was installed without OpenGL support. See https://github.com/openai/go-vncdriver for details on how debug.```

That means that your `go-vncdriver` was built without OpenGL
support. (The installer will first try to install with OpenGL, but
will fall back to installing without it.)

To figure out what happened, the easiest approach is to clone this
window and run `./build.sh`, which should print out the error upon
installing with OpenGL:

```
git clone https://github.com/openai/go-vncdriver.git
cd go-vncdriver
./build.sh
```

If you get errors like below then you need to install the [rendering dependencies](https://github.com/openai/go-vncdriver#rendering):
```
fatal error: X11/Xcursor/Xcursor.h: No such file or directory
fatal error: X11/extensions/Xrandr.h: No such file or directory
fatal error: X11/extensions/XInput.h: No such file or directory
fatal error: GL/gl.h: No such file or directory
```

Once you've fixed the issue, you should reinstall `go-vncdriver` via
`pip install --ignore-installed go-vncdriver`.

## Installation

By default, `go_vncdriver` will try to include OpenGL rendering. If
that build fails, it will fall back to omitting OpenGL rendering. (You
probably don't care about OpenGL rendering on a server anyway.)

### Dependencies

To install on Ubuntu 16.04, you need to have the following packages
installed:

```
sudo apt-get install -y python-dev make golang libjpeg-turbo8-dev
```

Same on Ubuntu 14.04, but the `golang` package is too old. Follow the installation instruction's on the [Go website](https://golang.org/doc/install) instead.

On OSX, the following should suffice:

```
brew install libjpeg-turbo golang
```

(On OSX newer than El Capitan, you may need to
[install golang](https://golang.org/doc/install) from their site, and
then just install `brew install libjpeg-turbo`.)

#### Rendering

The best way to see what your agent sees is to enable OpenGL
rendering. You'll need X and OpenGL development headers. On Ubuntu,
this is:

```
sudo apt-get install libx11-dev libxcursor-dev libxrandr-dev \
  libxinerama-dev libxi-dev libxxf86vm-dev libgl1-mesa-dev \
  mesa-common-dev
```

## Python versions

`go_vncdriver` has been tested on Python 2.7 and 3.5.
