# go-vncdriver

A fast VNC driver.

## OpenGL viewer

If you get an error of the form:

```go_vncdriver was installed without OpenGL support. See https://github.com/openai/go-vncdriver for details on how debug.```

That means that your `go-vncdriver` was built without OpenGL
support. (The installer will first try to install with OpenGL, but
will fall back to installing without it.)

To figure out what happened, the easiest approach is to clone this
repo and run `./build.py`, which should print out the error upon
installing with OpenGL:
```
git clone https://github.com/openai/go-vncdriver.git
cd go-vncdriver
./build.py
```
(NB if you're trying to compile for python 3.x
and your python binary is called python3 you'll need to modify the 
first line of build.py to the following otherwise the resulting
.so file will only work with python 2.x
```
#!/usr/bin/env python3
```

If you get errors like below then you need to install the [rendering dependencies](https://github.com/openai/go-vncdriver#rendering):
```
fatal error: X11/Xcursor/Xcursor.h: No such file or directory
fatal error: X11/extensions/Xrandr.h: No such file or directory
fatal error: X11/extensions/XInput.h: No such file or directory
fatal error: GL/gl.h: No such file or directory
```

Once you've fixed the issue, you should reinstall `go-vncdriver` by running the following command from `go-vncdriver` folder:
```
pip install -e ./
```

## Installation

### Dependencies

On Ubuntu 16.04:

```
sudo apt-get install -y python-dev make golang libjpeg-turbo8-dev
```

On Ubuntu 14.04:

```
sudo add-apt-repository ppa:ubuntu-lxc/lxd-stable  # for newer golang
sudo apt-get update
sudo apt-get install -y python-dev make golang libjpeg-turbo8-dev
```

On OSX:

```
brew install libjpeg-turbo golang
```

(On OSX newer than El Capitan, you may need to
[install golang](https://golang.org/doc/install) from their site, and
then just install `brew install libjpeg-turbo`.)

#### Rendering

OpenGL rendering is optional, but it's the best way to see what your
agent sees.

To enable it, you'll need X and OpenGL development headers.

On Ubuntu, this is:

```
sudo apt-get install libx11-dev libxcursor-dev libxrandr-dev \
  libxinerama-dev libxi-dev libxxf86vm-dev libgl1-mesa-dev \
  mesa-common-dev
```

## Python versions

`go_vncdriver` has been tested on Python 2.7 and 3.5.
