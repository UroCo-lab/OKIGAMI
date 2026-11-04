package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSleepDelaySettingsPersistAndRejectInvalid(t *testing.T) {
	b := testBoard(t)
	for _, seconds := range []int{0, 5, 30, 60, 180, 300} {
		body, _ := json.Marshal(map[string]int{"sleepAfterSeconds": seconds})
		if w := request(b, "POST", "/api/settings", b.token, body); w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
		restored := testBoard(t)
		restored.dir = b.dir
		restored.loadSleepSetting()
		if restored.sleepAfterSeconds != seconds {
			t.Fatalf("setting not restored: %d", restored.sleepAfterSeconds)
		}
	}
	for _, body := range []string{`{}`, `{"sleepAfterSeconds":-1}`, `{"sleepAfterSeconds":31}`, `{"sleepAfterSeconds":900}`, `{"sleepAfterSeconds":"30"}`} {
		if w := request(b, "POST", "/api/settings", b.token, []byte(body)); w.Code != 400 {
			t.Fatal(body, w.Code)
		}
		restored := testBoard(t)
		restored.dir = b.dir
		restored.loadSleepSetting()
		if b.sleepAfterSeconds != 300 || restored.sleepAfterSeconds != 300 {
			t.Fatal("invalid request changed saved delay")
		}
	}
	if err := os.WriteFile(filepath.Join(b.dir, "settings.json"), []byte(`{"sleepAfterSeconds":-10}`), 0600); err != nil {
		t.Fatal(err)
	}
	b.sleepAfterSeconds = 0
	b.loadSleepSetting()
	if b.sleepAfterSeconds != 0 {
		t.Fatal("invalid saved setting accepted")
	}
}

func TestIdleReadDoesNotExtendSleepButActivityDoes(t *testing.T) {
	b := testBoard(t)
	now := time.Now()
	b.sleepAfterSeconds = 30
	b.lastActivity = now.Add(-31 * time.Second)
	for _, path := range []string{"/api/config", "/api/current"} {
		request(b, "GET", path, b.token, nil)
	}
	if !b.autoSleepDue(now) {
		t.Fatal("read-only requests postponed sleep")
	}
	if w := request(b, "POST", "/api/activity", b.token, nil); w.Code != 204 {
		t.Fatal(w.Code)
	}
	if b.autoSleepDue(time.Now()) {
		t.Fatal("editing did not postpone sleep")
	}
	before := b.lastActivity
	b.sleeping = true
	if w := request(b, "POST", "/api/activity", b.token, nil); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if b.lastActivity != before || !b.sleeping {
		t.Fatal("activity modified sleeping device")
	}
}

func TestAutoSleepOnlyWhenIdleAndReady(t *testing.T) {
	now := time.Now()
	b := testBoard(t)
	b.sleepAfterSeconds = 5
	b.lastActivity = now.Add(-5 * time.Second)
	if !b.autoSleepDue(now) {
		t.Fatal("sleep deadline missed")
	}
	if b.autoSleepDue(now.Add(-time.Nanosecond)) {
		t.Fatal("slept early")
	}
	for _, flag := range []*bool{&b.loading, &b.sleeping, &b.closing, &b.sleepPending} {
		*flag = true
		if b.autoSleepDue(now) {
			t.Fatal("sleep requested in an unavailable state")
		}
		*flag = false
	}
	b.sleepAfterSeconds = 0
	if b.autoSleepDue(now) {
		t.Fatal("Kindle default scheduled custom sleep")
	}
	b.sleepAfterSeconds = 30
	b.lastActivity = time.Time{}
	if b.autoSleepDue(now) {
		t.Fatal("uninitialized deadline scheduled sleep")
	}
}

func TestSleepSettingSaveFailureKeepsPreviousValue(t *testing.T) {
	b := testBoard(t)
	b.sleepAfterSeconds = 60
	if err := os.Mkdir(filepath.Join(b.dir, "settings.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if w := request(b, "POST", "/api/settings", b.token, []byte(`{"sleepAfterSeconds":30}`)); w.Code != 500 {
		t.Fatal(w.Code)
	}
	if b.sleepAfterSeconds != 60 {
		t.Fatal("failed save changed active delay")
	}
}
