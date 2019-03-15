#!/bin/bash
#======================================================================================================#
# Copyright 2019 Trail of Bits. All rights reserved.
# filter.sh
#
# This script applies some typical "filters" to the output of Go's race detector.  This script expects
# its input to be in the form produced by normalize.py (a script accompanying this one).
#
# Specifically, this script removes the following entries:
#   * those containing a "failed to restore the stack" message,
#   * those involving the fmt package (such entries tend to be uninteresting), and
#   * those involving a goroutine that was "finished" when the race occurred.
#
# Finally, this script limits the entries to:
#   * those involving "onedge.WrapFunc", as all other entries would not have been produced by OnEdge.
#======================================================================================================#

set -eu

if [[ $# -ne 0 ]]; then
  echo "$0: expect no arguments" >&2
  exit 1
fi

cat \
| grep -v '\<failed to restore the stack\>' \
| grep -v '\<fmt\.' \
| grep -v '\<Goroutine (finished)' \
| grep '\<created at: onedge\.WrapFuncR\>'

#======================================================================================================#
