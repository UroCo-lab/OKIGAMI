package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLandscapeRotationRoundTrip(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 12, 8))
	img.SetGray(1, 2, color.Gray{Y: 51})
	img.SetGray(10, 6, color.Gray{Y: 170})
	rotated := rotateToPortrait(img, true)
	if rotated.Bounds().Dx() != 8 || rotated.Bounds().Dy() != 12 {
		t.Fatal(rotated.Bounds())
	}
	restored := rotateToLogical(rotated, true)
	if !bytes.Equal(img.Pix, restored.Pix) {
		t.Fatal("rotation changed pixels")
	}
}

func TestLandscapeUploadAndSleepStatus(t *testing.T) {
	b := testBoard(t)
	w := request(b, "POST", "/api/frame", b.token, frame(t, 1448, 1072))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !b.landscape {
		t.Fatal("orientation not saved")
	}
	data, err := os.ReadFile(filepath.Join(b.dir, "current.png"))
	if err != nil {
		t.Fatal(err)
	}
	awake, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	sleepData, err := os.ReadFile(filepath.Join(b.dir, "sleep", "board.png"))
	if err != nil {
		t.Fatal(err)
	}
	asleep, err := png.Decode(bytes.NewReader(sleepData))
	if err != nil {
		t.Fatal(err)
	}
	if awake.Bounds() != image.Rect(0, 0, 1072, 1448) || asleep.Bounds() != awake.Bounds() {
		t.Fatal("physical panel dimensions changed")
	}
	if bytes.Equal(data, sleepData) {
		t.Fatal("sleep status does not differ")
	}
	// Sleep removes both QR and icons without changing the rest of the content.
	a := rotateToLogical(awake, true)
	s := rotateToLogical(asleep, true)
	rect := toolbarRect(1448, 1072)
	for y := 0; y < 1072; y++ {
		for x := 0; x < 1448; x++ {
			if !image.Pt(x, y).In(rect) && !image.Pt(x, y).In(connectionQRRect(1448, 1072)) && a.GrayAt(x, y) != s.GrayAt(x, y) {
				t.Fatal("sleep changed user content")
			}
		}
	}
}

func TestCloseAndQRControlsAreDistinctInBothOrientations(t *testing.T) {
	b := testBoard(t)
	if b.controlHit(1008, 1380, 1072, 1448) != 2 || b.controlHit(116, 1332, 1072, 1448) != 1 || b.controlHit(912, 1380, 1072, 1448) != 0 || b.controlHit(500, 1380, 1072, 1448) != 0 {
		t.Fatal("portrait controls")
	}
	b.landscape = true
	// Close stays at the right; QR is logical lower-left (116,956).
	if b.controlHit(67, 1384, 1072, 1448) != 2 || b.controlHit(115, 116, 1072, 1448) != 1 || b.controlHit(67, 1288, 1072, 1448) != 0 {
		t.Fatal("landscape controls")
	}
}

func TestQRHitRegionFollowsOrientation(t *testing.T) {
	b := testBoard(t)
	if !b.buttonHit(116, 1332, 1072, 1448) || b.buttonHit(100, 100, 1072, 1448) {
		t.Fatal("portrait hit region")
	}
	b.landscape = true
	// Landscape logical (116,956) maps to physical (115,116).
	if !b.buttonHit(115, 116, 1072, 1448) || b.buttonHit(900, 200, 1072, 1448) {
		t.Fatal("landscape hit region")
	}
}

func TestSleepingBoardRejectsDisplayMutation(t *testing.T) {
	b := testBoard(t)
	b.sleeping = true
	for _, path := range []string{"/api/frame", "/api/qr"} {
		if got := request(b, "POST", path, b.token, frame(t, 1072, 1448)).Code; got != 409 {
			t.Fatalf("%s: %d", path, got)
		}
	}
}
