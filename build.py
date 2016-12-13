#!/usr/bin/env python

# Builds against your current Python version. You must have numpy installed.

from __future__ import print_function
import distutils.sysconfig
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
    os.chdir(os.path.dirname(os.path.abspath(__file__)))

    # Clear .build
    if os.path.exists('.build'):
        shutil.rmtree('.build')

    env = {}

    # Set up temporary GOPATH
    os.makedirs(os.path.normpath('.build/src/github.com/openai'))
    os.symlink('../../../..', '.build/src/github.com/openai/go-vncdriver')
    env['GOPATH'] = os.path.join(os.getcwd(), '.build')
    env['GO15VENDOREXPERIMENT'] = '1' # Needed on Go 1.5, no-op on Go 1.6+

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
        try:
            output = subprocess.check_output(['ld', '-ljpeg', '--trace-symbol', 'jpeg_CreateDecompress', '-e', '0'], stderr=subprocess.STDOUT)
            libjpg = output.decode().split(':')[0]
        except (subprocess.CalledProcessError, OSError):
            raise BuildException("Could not find libjpeg. HINT: try 'sudo apt-get install libjpeg-turbo8-dev' on Ubuntu or 'brew install libjpeg-turbo' on OSX")

    numpy_include = numpy.get_include()
    py_include = distutils.sysconfig.get_python_inc()
    plat_py_include = distutils.sysconfig.get_python_inc(plat_specific=1)
    includes = [numpy_include, py_include]
    if plat_py_include != py_include:
        includes.append(plat_py_include)
    env['CGO_CFLAGS'] = '-I' + ' -I'.join(includes)

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

    env['CGO_LDFLAGS'] = ' '.join([libjpg, ldflags])

    def build_no_gl():
        cmd = 'go build -tags no_gl -buildmode=c-shared -o go_vncdriver.so github.com/openai/go-vncdriver'
        eprint('Building without OpenGL: GOPATH={} {}'.format(os.getenv('GOPATH'), cmd))
        if subprocess.call(cmd.split()):
            raise BuildException('''
Build failed. HINT:

- Ensure you have your Python development headers installed. (On Ubuntu,
  this is just 'sudo apt-get install python-dev'.
''')

    def build_gl():
        cmd = 'go build -buildmode=c-shared -o go_vncdriver.so github.com/openai/go-vncdriver'
        eprint('Building with OpenGL: GOPATH={} {}. (Set GO_VNCDRIVER_NOGL to build without OpenGL.)'.format(os.getenv('GOPATH'), cmd))
        return not subprocess.call(cmd.split())

    eprint('Env info:\n')
    for k, v in env.items():
        eprint("export {}='{}'".format(k, v))
        os.environ[k] = v
    eprint()

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
