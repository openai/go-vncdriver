# go-vncdriver

A fast VNC driver.

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

To install on Ubuntu 16.04, you need to have the following packages
installed:

```
sudo apt-get install -y python-dev make golang libjpeg-turbo8-dev
```

On OSX, the following should suffice:

```
brew install golang libjpeg-turbo
```

If you'd also like to enable OpenGL, you'll need X and OpenGL
development headers. On Ubuntu, this is:

```
sudo apt-get install libx11-dev libxcursor-dev libxrandr-dev \
  libxinerama-dev libxi-dev libxxf86vm-dev libgl1-mesa-dev \
  mesa-common-dev
```

## Python versions

`go_vncdriver` has been tested on Python 2.7 and 3.5.
