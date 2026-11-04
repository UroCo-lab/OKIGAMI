#!/bin/sh
set -eu
BASE=/mnt/us/kindle-board
STATE="$BASE/state"
IMAGE="${2:-}"
export DISPLAY=:0
if [ "${1:-}" = activity ]; then
    POWER_STATE="$(lipc-get-prop com.lab126.powerd state 2>/dev/null)"
    if [ "$POWER_STATE" = active ]; then
        lipc-set-prop -i com.lab126.powerd touchScreenSaverTimeout 1
    fi
    exit 0
fi
case "$IMAGE" in "$STATE"/*.png) ;; *) exit 2 ;; esac
[ -f "$IMAGE" ] || exit 3
case "${1:-}" in
    sleep)
        POWER_STATE="$(lipc-get-prop com.lab126.powerd state 2>/dev/null)"
        [ "$POWER_STATE" != active ] || lipc-set-prop -i com.lab126.powerd powerButton 1
        exit 0
        ;;
    sleep-refresh)
        # Go explicitly paints black, white, then the posted image through X11.
        # Wait for each flashing update before sending the next frame.
        "$BASE/ss/bin/fbink_hf" -f -w -W GC16 -s
        exit 0
        ;;
    display|qr) ;;
    *) exit 2 ;;
esac
# Go paints the X11 backing pixmap first. Only request an e-ink refresh here;
# writing pixels directly to the framebuffer would desynchronize X11 again.
"$BASE/ss/bin/fbink_hf" -f -w -W GC16 -s
