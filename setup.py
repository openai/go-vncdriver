import os
import re
import subprocess
import sys

from distutils.command.build import build as DistutilsBuild
from setuptools import setup

def here():
    return os.path.join('.', os.path.dirname(__file__))

class BuildError(Exception):
    pass

class Build(DistutilsBuild):
    def run(self):
        self.check_version()
        self.build()

    def check_version(self):
        cmd = ['go', 'help', 'build']
        try:
            build_help = subprocess.check_output(cmd, stderr=subprocess.STDOUT).rstrip()
        except OSError as e:
            raise BuildError("""

Unable to execute '{}'. HINT: are you sure `go` is installed?

You can obtain a recent Go build from https://golang.org/doc/install. If on Ubuntu, you can follow: https://github.com/golang/go/wiki/Ubuntu. go_vncdriver requires at least Go 1.5.

(DETAIL: original error: {}.)""".format(' '.join(cmd), e))
        else:
            if 'buildmode' not in str(build_help):
                raise RuntimeError("""

Your Go installation looks too old: go_vncdriver requires at least Go 1.5.

You can obtain a recent Go build from https://golang.org/doc/install. If on Ubuntu, you can follow: https://github.com/golang/go/wiki/Ubuntu.

(DETAIL: the output of 'go help build' did not include 'buildmode'.)
""")

    def build(self):
        cmd = ['make', 'build']
        try:
            subprocess.check_call(cmd, cwd=here())
        except subprocess.CalledProcessError as e:
            sys.stderr.write("Could not build go_vncdriver: %s\n" % e)
            raise
        except OSError as e:
            raise BuildError("Unable to execute '{}'. HINT: are you sure `make` is installed? (original error: {}.)".format(' '.join(cmd), e))
        DistutilsBuild.run(self)

setup(name='go_vncdriver',
      version='0.4.4',
      cmdclass={'build': Build},
      packages=['go_vncdriver'],
      package_dir={'go_vncdriver': '.'},
      package_data={'go_vncdriver': ['go_vncdriver.so']},
      setup_requires=['numpy'],
)
