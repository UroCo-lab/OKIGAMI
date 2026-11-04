#!/bin/sh
# Name: OKIGAMI
# Author: Local Tools
# DontUseFBInk
BASE=/mnt/us/kindle-board
if [ -f "$BASE/run.sh" ]; then
    exec sh "$BASE/run.sh" start >/dev/null 2>&1
fi
