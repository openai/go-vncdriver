# go-vncdriver

A fast VNC driver.

## Installation on Ubuntu:

If you have Ubuntu 14, get the latest Go compiler:
```sh
# Needed for Ubuntu 14, not Ubuntu 16
$ sudo add-apt-repository ppa:ubuntu-lxc/lxd-stable  # for newer golang
$ sudo apt-get update
```
Then
```sh
$ sudo apt-get install -y python-dev make golang libjpeg-turbo8-dev
```
And if you want OpenGL rendering support (you probably do, unless you're running on a headless server):
```sh
$ sudo apt-get install libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev \
  libxxf86vm-dev libgl1-mesa-dev mesa-common-dev
```
NOTE: If you're using a Python named something other than `python`, such as `python3`, replace both `python` and `pip` below with the commands for the corresponding Python
```sh
$ git clone https://github.com/openai/go-vncdriver.git
$ cd go-vncdriver
$ python build.py
$ pip install -e .
```

## Installation on OSX:

```
$ brew install libjpeg-turbo golang
```

(On OSX newer than El Capitan, you may need to
[install golang](https://golang.org/doc/install) from their site, and
then just install `brew install libjpeg-turbo`.)

Then
```
$ git clone https://github.com/openai/go-vncdriver.git
$ cd go-vncdriver
$ python build.py
$ pip install -e .
```

## OpenGL viewer

The OpenGL renderer is optional. If you get an error of the form:

```
go_vncdriver was installed without OpenGL support. See https://github.com/openai/go-vncdriver for details on how debug.
```

That means that your `go-vncdriver` was built without OpenGL
support. (The installer will first try to install with OpenGL, but
will fall back to installing without it.)

Do the installation steps above, including the extra dependencies to add OpenGL rendering.

If you get errors like below, the dependencies aren't installed properly:
```
fatal error: X11/Xcursor/Xcursor.h: No such file or directory
fatal error: X11/extensions/Xrandr.h: No such file or directory
fatal error: X11/extensions/XInput.h: No such file or directory
fatal error: GL/gl.h: No such file or directory
```

## Python versions

`go_vncdriver` has been tested on Python 2.7 and 3.5.
