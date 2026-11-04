package main

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"strings"
	"time"
)

func (b *board) watchPower(ctx context.Context) {
	if b.hook == "" {
		return
	}
	cmd := exec.CommandContext(ctx, "lipc-wait-event", "-m", "com.lab126.powerd", "goingToScreenSaver,outOfScreenSaver,goingToPasswdDlg")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		b.reportError("power events", err)
		return
	}
	if err = cmd.Start(); err != nil {
		b.reportError("power events", err)
		return
	}
	go func() {
		defer cmd.Wait()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("power event: %s", line)
			if strings.Contains(line, "goingToScreenSaver") {
				b.mu.Lock()
				b.sleeping = true
				b.sleepPending = false
				started := time.Now()
				if err := b.refreshSleepDisplay(func(path string) error { return b.action("sleep-refresh", path) }); err != nil {
					b.reportError("sleep redraw", err)
				} else {
					log.Printf("sleep cleaning completed in %s", time.Since(started))
				}
				b.mu.Unlock()
			}
			if strings.Contains(line, "goingToPasswdDlg") {
				b.mu.Lock()
				b.sleeping = true
				if b.nativeShow != nil {
					b.nativeShow(false)
				}
				b.mu.Unlock()
			}
			if strings.Contains(line, "outOfScreenSaver") {
				go func() {
					// Allow the stock wake transition to finish, then repaint our backing pixmap.
					select {
					case <-ctx.Done():
						return
					case <-time.After(200 * time.Millisecond):
					}
					b.mu.Lock()
					defer b.mu.Unlock()
					if b.closing {
						return
					}
					b.sleeping = false
					b.sleepPending = false
					if b.nativeShow != nil {
						b.nativeShow(true)
					}
					if err := b.showCurrent(); err != nil {
						b.reportError("wake redraw", err)
					}
					b.noteActivityLocked(time.Now())
				}()
			}
		}
	}()
}
