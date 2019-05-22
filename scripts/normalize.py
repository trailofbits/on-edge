#!/usr/bin/env python3.7
#======================================================================================================#
# Copyright 2019 Trail of Bits
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.# Makefile for OnEdge
#======================================================================================================#
# This script "normalizes" the output of Go's race detector, e.g., to make it easier to compare such
# output across multiple runs of a program.
#
# Each of the race detector's reports looks something like the following.
#
#   ==================
#   WARNING: DATA RACE
#   Read at 0x<address A> by goroutine I:
#     <stack trace S>
#
#   Previous write at 0x<address B> by goroutine J:
#     <stack trace T>
#
#   Goroutine I (running) created at:
#     <stack trace U>
#
#   Goroutine J (finished) created at:
#     <stack trace V>
#   ==================
#
# This script would turn the above into four tab separated fields with the following form.
#
#   Write at 0x<address B> by: <stack trace T>
#   Read at 0x<address A> by: <stack trace S>
#   Goroutine (finished) created at: <stack trace V>
#   Goroutine (running) created at: <stack trace U>
#
# Details of how this is done are given in the comments below.
#
# You might use this script as follows.
#
#   $ program_under_test 2>&1 1>/dev/null | normalize.py
#
#======================================================================================================#

import re
import sys

if len(sys.argv) != 1:
  sys.stderr.write("%s: expect no arguments\n" % sys.argv[0])
  sys.exit(1)

#======================================================================================================#

def normalize_and_print_fields(fields):
  assert len(fields) == 4
  
  # sam.moelius: Remove "Previous " and capitalize first letter of "read" or "write".
  assert re.match(r'^Previous [rw]', fields[1])
  fields[1] = fields[1][9:]
  fields[1] = fields[1][0].upper() + fields[1][1:]
  
  # sam.moelius: Ensure first field begins with "Write".
  if re.match(r'^Read ', fields[0]):
    tmp = fields[0]
    fields[0] = fields[1]
    fields[1] = tmp
    tmp = fields[2]
    fields[2] = fields[3]
    fields[3] = tmp
  assert re.match(r'^Write ', fields[0])

  # sam.moelius: Remove goroutine ids.
  fields[0] = re.sub(r'\b(by) goroutine [0-9]+(:)', r'\1\2', fields[0])
  fields[1] = re.sub(r'\b(by) goroutine [0-9]+(:)', r'\1\2', fields[1])
  fields[2] = re.sub(r'^(Goroutine) [0-9]+( )', r'\1\2', fields[2])
  fields[3] = re.sub(r'^(Goroutine) [0-9]+( )', r'\1\2', fields[3])

  # sam.moelius: Print fields.
  for i in range(len(fields)):
    if i > 0:
      print("\t", end="")
    print("%s" % fields[i], end="")
  print("")

#======================================================================================================#

in_data_race = False
fields = []
field = ""

for line in sys.stdin:
  line = line.strip()
  if not in_data_race:
    if re.match(r'^=+$', line):
      fields = []
      in_data_race = True
  else:
    if re.match(r'^=+$', line):
      fields.append(field)
      field = ""
      normalize_and_print_fields(fields)
      in_data_race = False
    elif line == "WARNING: DATA RACE":
      continue
    elif line != "":
      if field != "":
        field += " "
      field += line
    else:
      fields.append(field)
      field = ""

#======================================================================================================#
