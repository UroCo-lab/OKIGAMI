#!/bin/sh
# Refuse to signal a recycled PID belonging to an unrelated process.
ACTION="$1"
PIDFILE="$2"
EXPECTED="$3"
PID="$(cat "$PIDFILE" 2>/dev/null)"
case "$PID" in ''|*[!0-9]*) exit 1 ;; esac
kill -0 "$PID" 2>/dev/null || exit 1
[ -r "/proc/$PID/cmdline" ] || exit 1
tr '\000' ' ' < "/proc/$PID/cmdline" | grep -F "$EXPECTED" >/dev/null 2>&1 || exit 1
case "$ACTION" in
    alive) exit 0 ;;
    stop) kill "$PID" ;;
    interrupt) kill -INT "$PID" ;;
    *) exit 2 ;;
esac
