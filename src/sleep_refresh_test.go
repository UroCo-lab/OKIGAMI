package main

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSleepCleaningFramesPreservePhoto(t *testing.T) {
	b := testBoard(t)
	if w := request(b, "POST", "/api/frame", b.token, frame(t, 1072, 1448)); w.Code != 200 {
		t.Fatal(w.Code)
	}
	saved, err := os.ReadFile(filepath.Join(b.dir, "sleep", "board.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = b.prepareSleepRefresh(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		value byte
	}{{"refresh-black.png", 0}, {"refresh-white.png", 255}} {
		data, err := os.ReadFile(filepath.Join(b.dir, tc.name))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 1072 || img.Bounds().Dy() != 1448 {
			t.Fatal("cleaning frame dimensions")
		}
		for y := 0; y < 1448; y++ {
			for x := 0; x < 1072; x++ {
				r, _, _, _ := img.At(x, y).RGBA()
				if byte(r>>8) != tc.value {
					t.Fatal("cleaning frame is not solid")
				}
			}
		}
	}
	after, err := os.ReadFile(filepath.Join(b.dir, "sleep", "board.png"))
	if err != nil || !bytes.Equal(saved, after) {
		t.Fatal("cleaning frames replaced posted photo")
	}
}

func TestSleepCleaningAlwaysRestoresPostedImageLast(t *testing.T) {
	b := testBoard(t)
	failure := errors.New("intermediate update failed")
	var called []string
	err := b.refreshSleepDisplay(func(path string) error {
		called = append(called, path)
		if len(called) == 1 {
			return failure
		}
		return nil
	})
	want := []string{filepath.Join(b.dir, "refresh-black.png"), filepath.Join(b.dir, "refresh-white.png"), filepath.Join(b.dir, "sleep", "board.png")}
	if !reflect.DeepEqual(called, want) || !errors.Is(err, failure) {
		t.Fatal("posted image was not restored after intermediate failure", called, err)
	}
}
