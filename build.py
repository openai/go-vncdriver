#!/usr/bin/env python

# Builds against your current Python version. You must have numpy installed.

from __future__ import print_function
import os
import numpy
import re
import shutil
import subprocess
import sys
import sysconfig

class BuildException(Exception):
    pass

def main():
    try:
        build()
    except BuildException as e:
        sys.exit(e)

def build():
    os.chdir(os.path.dirname(__file__))

    # Clear .build
    try:
        shutil.rmtree('.build')
    except FileNotFoundError:
        pass

    os.makedirs(os.path.normpath('.build/src/github.com/openai'))
    os.symlink('../../../..', '.build/src/github.com/openai/go-vncdriver')
    os.symlink('../../../vendor/github.com/go-gl', '.build/src/github.com/go-gl')
    os.symlink('../../../vendor/github.com/op', '.build/src/github.com/op')
    os.symlink('../../../vendor/github.com/juju', '.build/src/github.com/juju')
    os.symlink('../../../vendor/github.com/pixiv', '.build/src/github.com/pixiv')

    # We need to prevent from linking against Anaconda Python's libpjpeg.
    #
    # Right now we hardcode these paths, and fall back to actually looking
    # it up.
    libjpg = os.getenv('LIBJPG')
    if not libjpg:
        for i in ['/usr/lib/x86_64-linux-gnu/libjpeg.so', '/usr/local/opt/jpeg-turbo/lib/libjpeg.dylib']:
            if os.path.exists(i):
                libjpg = i
                break

    # Note this might mean not getting libjpeg-turbo, which is quite nice
    # to have. Also, it doesn't work on macOS.
    if not libjpg:
        result = subprocess.run(['ld', '-ljpeg', '--trace-symbol', 'jpeg_CreateDecompress', '-e', '0'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        if not result.returncode:
            libjpg = result.stdout.decode().split(':')[0]

    if not libjpg:
        raise BuildException("Could not find libjpeg. HINT: try 'sudo apt-get install libjpeg-turbo8-dev' on Ubuntu or 'brew install libjpeg-turbo' on OSX")

    os.environ['GOPATH'] = os.path.join(os.getcwd(), '.build')
    os.environ['CGO_CFLAGS'] = '-I{} -I{}'.format(
            numpy.get_include(), sysconfig.get_config_var('INCLUDEPY'))

    if os.uname()[0] == 'Darwin':
        # Don't link to libpython, since some installs only have a static version available,
        # and statically linking libpython doesn't work for a C extension -- it will duplicate
        # all the global variables, among other things.
        #
        # Instead, just leave Python symbols undefined and let the loader resolve them
        # at runtime. TODO(jeremy): We might want this behavior on Linux, too.
        #
        # In Darwin, ld returns an error by default on undefined symbols. Use dynamic_lookup instead.
        ldflags = '-undefined dynamic_lookup'
    else:
        library = sysconfig.get_config_var('LIBRARY')
        match = re.search('^lib(.*)\.a', library)
        if match is None:
          raise BuildException('Could not parse LIBRARY: {}'.format(library))
        ldflags = '-L{} -l{}'.format(sysconfig.get_config_var('LIBDIR'), match.group(1))

    os.environ['CGO_LDFLAGS'] = ' '.join([libjpg, ldflags])

    os.chdir(os.path.normpath(os.getcwd() + '/.build/src/github.com/openai/go-vncdriver'))

    def build_no_gl():
        cmd = 'go build -tags no_gl -buildmode=c-shared -o go_vncdriver.so'
        eprint('Building without OpenGL: GOPATH={} {}'.format(os.getenv('GOPATH'), cmd))
        if subprocess.run(cmd.split()).returncode:
            raise BuildException('''
Build failed. HINT:

- Ensure you have your Python development headers installed. (On Ubuntu,
  this is just 'sudo apt-get install python-dev'.
''')

    def build_gl():
        cmd = 'go build -buildmode=c-shared -o go_vncdriver.so'
        eprint('Building with OpenGL: GOPATH={} {}. (Set GO_VNCDRIVER_NOGL to build without OpenGL.)'.format(os.getenv('GOPATH'), cmd))
        return not subprocess.run(cmd.split()).returncode

    eprint('Env info:\n\nexport CGO_LDFLAGS={}\nexport CGO_CFLAGS={}\n'.format(os.getenv('CGO_LDFLAGS'), os.getenv('CGO_CFLAGS')))

    if os.getenv('GO_VNCDRIVER_NOGL'):
        build_no_gl()
    else:
        eprint('Running build with OpenGL rendering.')
        if not build_gl():
            eprint('\nNote: could not build with OpenGL rendering (cf https://github.com/openai/go-vncdriver). This is expected on most servers. Going to try building without OpenGL.')
            build_no_gl()

# http://stackoverflow.com/a/14981125
def eprint(*args, **kwargs):
    print(*args, file=sys.stderr, **kwargs)

if __name__ == '__main__':
    main()
