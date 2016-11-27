from __future__ import absolute_import
import sys

# We need to ensure this gets imported before tensorflow.
# This at least ensures we're not failing for import ordering reasons.
if 'tensorflow' in sys.modules:
    raise RuntimeError('go_vncdriver must be imported before tensorflow')

from go_vncdriver.go_vncdriver import *
