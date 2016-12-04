#!/bin/bash

# Helpful reference: https://github.com/pypa/python-manylinux-demo/blob/master/travis/build-wheels.sh

set -e -x

if [ ! -d /opt/libjpeg-turbo ]; then
    rpm --import /io/travis/LJT-GPG-KEY
    yum install -y /io/travis/libjpeg-turbo-official-1.5.1.$(uname -i).rpm
fi
export CPATH=/opt/libjpeg-turbo/include

if [ $(uname -i) = "x86_64" ]; then
    GOARCH=amd64
    export LIBJPG=/opt/libjpeg-turbo/lib64/libturbojpeg.a
else
    GOARCH=i386
    export LIBJPG=/opt/libjpeg-turbo/lib32/libturbojpeg.a
fi

if [ ! -d /usr/local/go ]; then
    tar -C /usr/local -xf /io/travis/go1.7.4.linux-$GOARCH.tar.gz
fi
export PATH=$PATH:/usr/local/go/bin

# Delete "-ljpeg" argument in cgo, since we want to link to a specific archive.
sed -i '/#cgo LDFLAGS: -ljpeg/d' io/vendor/github.com/pixiv/go-libjpeg/jpeg/jpeg.go

go run /io/travis/build-wheels.go
