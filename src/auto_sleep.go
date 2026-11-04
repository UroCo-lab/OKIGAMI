package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func validSleepDelay(seconds int) bool {
	switch seconds {
	case 0, 5, 30, 60, 180, 300:
		return true
	}
	return false
}
func (b *board) saveSleepSetting() error {
	data, _ := json.Marshal(map[string]int{"sleepAfterSeconds": b.sleepAfterSeconds})
	return os.WriteFile(filepath.Join(b.dir, "settings.json"), data, 0600)
}
func (b *board) loadSleepSetting() {
	data, err := os.ReadFile(filepath.Join(b.dir, "settings.json"))
	if err != nil {
		return
	}
	var settings struct {
		SleepAfterSeconds int `json:"sleepAfterSeconds"`
	}
	if json.Unmarshal(data, &settings) == nil && validSleepDelay(settings.SleepAfterSeconds) {
		b.sleepAfterSeconds = settings.SleepAfterSeconds
	}
}
func (b *board) noteActivityLocked(now time.Time) {
	if b.sleeping || b.closing {
		return
	}
	b.lastActivity = now
	if b.hook != "" && now.Sub(b.lastPowerTouch) >= 5*time.Second {
		b.lastPowerTouch = now
		if err := b.action("activity", ""); err != nil {
			b.reportError("reset native idle timeout", err)
		}
	}
}
func (b *board) autoSleepDue(now time.Time) bool {
	return b.sleepAfterSeconds > 0 && !b.lastActivity.IsZero() && !b.sleeping && !b.loading && !b.closing && !b.sleepPending && now.Sub(b.lastActivity) >= time.Duration(b.sleepAfterSeconds)*time.Second
}
func (b *board) watchAutoSleep(ctx context.Context) {
	if b.hook == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				b.mu.Lock()
				if b.autoSleepDue(now) {
					b.sleepPending = true
					if err := b.action("sleep", filepath.Join(b.dir, "sleep", "board.png")); err != nil {
						b.sleepPending = false
						b.lastActivity = now
						b.reportError("automatic sleep", err)
					}
				}
				b.mu.Unlock()
			}
		}
	}()
}
