#!/bin/sh

# Builds against your current Python version. You must have numpy installed.

set -eu

cd "$(dirname "$0")"

# Allow setup.py to override which python gets used
PYTHON=${GO_VNCDRIVER_PYTHON:-python}

# Clear .build
rm -rf .build
#trap 'rm -rf .build' EXIT

mkdir -p .build/src/github.com/openai
ln -s ../../../.. .build/src/github.com/openai/go-vncdriver
ln -s ../../../vendor/github.com/go-gl .build/src/github.com
ln -s ../../../vendor/github.com/op .build/src/github.com
ln -s ../../../vendor/github.com/juju .build/src/github.com
ln -s ../../../vendor/github.com/pixiv .build/src/github.com

export GOPATH="$(pwd)/.build"
export CGO_CFLAGS="$(
    ${PYTHON} -c "import numpy, sysconfig
print('-I{} -I{}\n'.format(numpy.get_include(), sysconfig.get_config_var('INCLUDEPY')))"
)"
if [ -z "$CGO_CFLAGS" ]; then
    echo "Could not populate CGO_CFLAGS (see error above)"
    exit 1
fi

if [[ $(uname) == 'Darwin' ]]; then
    # Don't link to libpython, since some installs only have a static version available,
    # and statically linking libpython doesn't work for a C extension -- it will duplicate
    # all the global variables, among other things.
    #
    # Instead, just leave Python symbols undefined and let the loader resolve them
    # at runtime. TODO(jeremy): We might want this behavior on Linux, too.
    #
    # In Darwin, ld returns an error by default on undefined symbols. Use dynamic_lookup instead.
    export CGO_LDFLAGS="-undefined dynamic_lookup"
else
    export CGO_LDFLAGS="$(${PYTHON} -c "import re, sysconfig;
library = sysconfig.get_config_var('LIBRARY')
match = re.search('^lib(.*)\.a', library)
if match is None:
  raise RuntimeError('Could not parse LIBRARY: {}'.format(library))
print('-L{} -l{}\n'.format(sysconfig.get_config_var('LIBDIR'), match.group(1)))
")"
fi
if [ -z "$CGO_LDFLAGS" ]; then
    echo "Could not populate CGO_LDFLAGS (see error above)"
    exit 1
fi

cd "$(pwd)/.build/src/github.com/openai/go-vncdriver"

build_no_gl() {
    echo >&2 "Building without OpenGL: go build -tags no_gl -buildmode=c-shared -o go_vncdriver.so"
    go build -tags no_gl -buildmode=c-shared -o go_vncdriver.so || (
	cat >&2 <<EOF
Build failed. HINT:

- Ensure you have your Python development headers installed. (On Ubuntu,
  this is just 'sudo apt-get install python-dev'.
EOF
	exit 1
    )
}

build_gl() {
    echo >&2 "Building with OpenGL: go build -buildmode=c-shared -o go_vncdriver.so. (Set GO_VNCDRIVER_NOGL to build without OpenGL.)"
    go build -buildmode=c-shared -o go_vncdriver.so
}

echo >&2 "Env info:"
echo >&2
echo >&2 "export CGO_LDFLAGS='${CGO_LDFLAGS}'"
echo >&2 "export CGO_CFLAGS='${CGO_CFLAGS}'"
echo >&2
if ! [ -z "${GO_VNCDRIVER_NOGL:-}" ]; then
    build_no_gl
else
    echo >&2 "Running build with OpenGL rendering."
    if ! build_gl; then
	echo >&2
	echo >&2 "Note: could not build with OpenGL rendering (cf https://github.com/openai/go-vncdriver). This is expected on most servers. Going to try building without OpenGL."

	build_no_gl
    fi
fi
