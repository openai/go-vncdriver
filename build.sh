#!/bin/sh

# Builds against your current Python version. You must have numpy installed.

set -e

cd "$(dirname "$0")"

# Clear .build
rm -rf .build
#trap 'rm -rf .build' EXIT

mkdir -p .build/src/github.com/openai
ln -s ../../../.. .build/src/github.com/openai/go-vncdriver
ln -s ../../../vendor/github.com/go-gl .build/src/github.com
ln -s ../../../vendor/github.com/op .build/src/github.com
ln -s ../../../vendor/github.com/juju .build/src/github.com
ln -s ../../../vendor/github.com/pixiv .build/src/github.com

# We need to prevent from linking against Anaconda Python's libpjpeg.
#
# Right now we hardcode these paths, and could fall back to actually
# looking it up. But this might mean not getting libjpeg-turbo, which
# is quite nice to have.
#
# Could do the lookup with:
#
# ld -ljpeg --trace-symbol jpeg_CreateDecompress -e 0
for i in /usr/lib/x86_64-linux-gnu/libjpeg.so /usr/local/opt/jpeg-turbo/lib/libjpeg.dylib; do
    if [ -e "$i" ]; then
	LIBJPG="$i"
	break
    fi
done

if [ -z "${LIBJPG:-}" ]; then
    echo >&2 "Could not find libjpeg. HINT: try 'sudo apt-get install libjpeg-turbo8-dev' on Ubuntu or 'brew install libjpeg-turbo' on OSX"
    exit 1
fi

export GOPATH="$(pwd)/.build"
export CGO_CFLAGS="$(
    python -c "import numpy, sysconfig
print('-I{} -I{}\n'.format(numpy.get_include(), sysconfig.get_config_var('INCLUDEPY')))"
)"
if [ -z "$CGO_CFLAGS" ]; then
    echo "Could not populate CGO_CFLAGS (see error above)"
    exit 1
fi

export CGO_LDFLAGS="${LIBJPG} $(python -c "import re, sysconfig;
library = sysconfig.get_config_var('LIBRARY')
match = re.search('^lib(.*)\.a', library)
if match is None:
  raise RuntimeError('Could not parse LIBRARY: {}'.format(library))
print('-L{} -l{}\n'.format(sysconfig.get_config_var('LIBDIR'), match.group(1)))
")" #"
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
if [ "$1" = "no_gl" ] || ! [ -z "${GO_VNCDRIVER_NOGL:-}" ]; then
    build_no_gl
else
    echo >&2 "Running build with OpenGL rendering."
    if ! build_gl; then
	echo >&2
	echo >&2 "Note: could not build with OpenGL rendering (cf https://github.com/openai/blob/master/go-vncdriver/README.md). This is expected on most servers. Going to try building without OpenGL."

	build_no_gl
    fi
fi
