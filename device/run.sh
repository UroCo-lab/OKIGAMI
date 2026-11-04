#!/bin/sh
BASE=/mnt/us/kindle-board
STATE="$BASE/state"
SERVER_PID=/tmp/kindle_board_server.pid
WORKER_PID=/tmp/kindle_board_worker.pid
SHIELD_PID=/tmp/kindle_board_receive_shield.pid
IPTABLES_RULE=''
export DISPLAY=:0
mkdir -p "$STATE"
STATEFILE=/tmp/kindle_board_native_renderers
LOG="$STATE/runtime.log"
. "$BASE/ss/blanket_renderers.sh"
log() { echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG"; }
return_to_library() {
    if lipc-set-prop com.lab126.appmgrd start 'app://com.lab126.KPPMainApp?view=KPP_LIBRARY' >>"$LOG" 2>&1; then
        log 'Library view requested'
    else
        lipc-set-prop com.lab126.appmgrd start 'app://com.lab126.booklet.home' >>"$LOG" 2>&1
    fi
    xrefresh
}

remove_rule() {
    if [ -f /tmp/kindle_board_firewall ]; then
        iptables -D INPUT -i wlan0 -p tcp --dport 8080 -j ACCEPT 2>/dev/null
        rm -f /tmp/kindle_board_firewall
    fi
}
receive_cleanup() {
    sh "$BASE/process.sh" stop "$SHIELD_PID" "$BASE/ss/bin/screensaver_shield" 2>/dev/null
    rm -f "$SHIELD_PID"
    remove_rule
    if [ -f /tmp/kindle_board_original_prevent ]; then
        PREVIOUS="$(cat /tmp/kindle_board_original_prevent)"
        case "$PREVIOUS" in 0|1) lipc-set-prop -i com.lab126.powerd preventScreenSaver "$PREVIOUS" ;; esac
        rm -f /tmp/kindle_board_original_prevent
    fi
    if restore_original_renderers; then rm -f "$STATEFILE"; fi
}
case "${1:-start}" in
    start)
        if sh "$BASE/process.sh" alive "$WORKER_PID" "$BASE/run.sh worker"; then
            sh "$BASE/process.sh" interrupt "$SERVER_PID" "$BASE/bin/kindle-board" 2>/dev/null
            sleep 2
            sh "$BASE/process.sh" alive "$WORKER_PID" "$BASE/run.sh worker" && exit 0
        fi
        chmod +x "$BASE/bin/kindle-board" "$BASE/ss/bin/"* "$BASE/"*.sh "$BASE/ss/"*.sh
        # SH_Integration waits for its script child. Keep that child alive while
        # the native window is open, so the launcher does not return home early.
        exec sh "$BASE/run.sh" worker >>"$STATE/session.log" 2>&1
        ;;
    worker)
        echo $$ > "$WORKER_PID"
        echo 'KINDLE_BOARD_START_0.6.0'
        date -u
        uname -srmo
        id -u
        # The server checks the runtime, QR encoder, and display on real startup.
        # Running a second full QR encoder self-test here only delays the UI.
        [ -x "$BASE/bin/kindle-board" ] && [ -x "$BASE/ss/bin/fbink_hf" ] || exit 1
        cleanup() {
            receive_cleanup
            rm -f "$SERVER_PID" "$WORKER_PID"
        }
        trap 'sh "$BASE/process.sh" interrupt "$SERVER_PID" "$BASE/bin/kindle-board" 2>/dev/null; cleanup; exit 0' HUP INT TERM
        trap 'cleanup' EXIT
        # Stop the previous independent renderer before acquiring the display.
        sh "$BASE/ss/toggle.sh" disable || exit 1
        capture_renderer_state || exit 1
        disable_original_renderers || exit 1
        IP="$(ifconfig wlan0 2>/dev/null | sed -n 's/.*inet addr:\([0-9.]*\).*/\1/p' | head -n 1)"
        if [ -z "$IP" ]; then
            "$BASE/ss/bin/fbink_hf" -p -m -M -S 3 -B WHITE -C BLACK -w 'Connect Kindle to Wi-Fi, then open OKIGAMI again.'
            sleep 3
            xrefresh
            exit 1
        fi
        if iptables -I INPUT -i wlan0 -p tcp --dport 8080 -j ACCEPT; then
            touch /tmp/kindle_board_firewall
        else
            echo 'Cannot open the local receiver port'
            exit 1
        fi
        "$BASE/bin/kindle-board" -data "$STATE" -hook "$BASE/display.sh" -listen ':8080' -ip "$IP" &
        echo $! > "$SERVER_PID"
        wait "$(cat "$SERVER_PID")"
        RESULT=$?
        receive_cleanup
        rm -f "$SERVER_PID"
        rm -f "$STATE/receiver-active"
        POWER_STATE="$(lipc-get-prop com.lab126.powerd state 2>/dev/null)"
        [ "$POWER_STATE" != active ] || return_to_library
        ;;
    stop)
        sh "$BASE/process.sh" interrupt "$SERVER_PID" "$BASE/bin/kindle-board" 2>/dev/null
        COUNT=0
        while sh "$BASE/process.sh" alive "$WORKER_PID" "$BASE/run.sh worker"; do
            sleep 1
            COUNT=$((COUNT+1))
            [ "$COUNT" -lt 8 ] || break
        done
        if sh "$BASE/process.sh" alive "$WORKER_PID" "$BASE/run.sh worker"; then
            sh "$BASE/process.sh" stop "$WORKER_PID" "$BASE/run.sh worker" 2>/dev/null
            sleep 1
        fi
        receive_cleanup
        sh "$BASE/ss/toggle.sh" disable
        rm -f "$STATE/sleep-request"
        if ! sh "$BASE/process.sh" alive "$SERVER_PID" "$BASE/bin/kindle-board"; then rm -f "$STATE/receiver-active"; fi
        return_to_library
        ;;
    *) exit 2 ;;
esac
