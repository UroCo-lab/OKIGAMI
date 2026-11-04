package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAwakeOverlayOnlyChangesQRAndIcons(t *testing.T) {
	b := testBoard(t)
	b.url = "http://192.0.2.10:8080/#" + strings.Repeat("a", 48)
	if err := b.ensureConnectionQR(); err != nil {
		t.Fatal(err)
	}
	for _, size := range []image.Point{{1072, 1448}, {1448, 1072}} {
		raw := image.NewGray(image.Rect(0, 0, size.X, size.Y))
		draw.Draw(raw, raw.Bounds(), image.NewUniform(color.Gray{Y: 136}), image.Point{}, draw.Src)
		awake := composeToolbar(raw, false, false, true, deviceStatus{Wifi: true, WifiBars: 3, Battery: 89}, b.smallQR)
		asleep := composeToolbar(raw, true, true, false, deviceStatus{}, b.smallQR)
		if !bytes.Equal(raw.Pix, asleep.Pix) {
			t.Fatal("sleep changed posted content")
		}
		if bytes.Equal(raw.Pix, awake.Pix) {
			t.Fatal("awake overlays were hidden by old setting")
		}
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				if !image.Pt(x, y).In(toolbarRect(size.X, size.Y)) && !image.Pt(x, y).In(connectionQRRect(size.X, size.Y)) && awake.GrayAt(x, y) != raw.GrayAt(x, y) {
					t.Fatal("overlay changed unrelated content")
				}
			}
		}
		if output := os.Getenv("KINDLE_BOARD_VISUAL_OUTPUT"); output != "" {
			name := "portrait"
			if size.X > size.Y {
				name = "landscape"
			}
			for suffix, img := range map[string]image.Image{"awake": awake, "sleep": asleep, "original": raw} {
				if err := writePNG(filepath.Join(output, "inline-qr-"+name+"-"+suffix+".png"), img); err != nil {
					t.Fatal(err)
				}
			}
			if err := writePNG(filepath.Join(output, "connection-qr.png"), b.smallQR); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(output, "qr-test-url.txt"), []byte(b.url), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestWirelessSignalReadings(t *testing.T) {
	for _, tc := range []struct {
		data string
		want int
	}{
		{"wlan0: 0000 70. -52. -96. 0 0 0", 3},
		{"wlan0: 0000 52. -68. -96. 0 0 0", 2},
		{"wlan0: 0000 29. -81. -96. 0 0 0", 1},
		{"wlan0: 0000 10. -95. -96. 0 0 0", 0},
		{"wlan0: 0000 0. 0. 0. 0 0 0", -1},
		{"wlan0: 0000 70. 65. 0. 0 0 0", -1},
		{"wlan1: 0000 70. -52. -96. 0 0 0", -1},
		{"wlan0: bad", -1},
		{"wlan0: 0000 70. -256. -96. 0 0 0", -1},
	} {
		if got := wirelessBars(tc.data); got != tc.want {
			t.Errorf("%q: %d, want %d", tc.data, got, tc.want)
		}
	}
	if iwconfigBars("wlan0 IEEE 802.11  Link Quality=62/70  Signal level=-48 dBm") != 3 || iwconfigBars("Signal level:-79 dBm") != 1 || iwconfigBars("Not-Associated") != -1 {
		t.Fatal("iwconfig fallback misread signal")
	}
}

func TestToolbarRightEdgeAndRoundedCorners(t *testing.T) {
	for _, size := range []image.Point{{1072, 1448}, {1448, 1072}} {
		raw := image.NewGray(image.Rect(0, 0, size.X, size.Y))
		draw.Draw(raw, raw.Bounds(), image.NewUniform(color.Gray{Y: 68}), image.Point{}, draw.Src)
		shown := composeToolbar(raw, false, true, false, deviceStatus{Wifi: true, WifiBars: 2, Battery: 42})
		if r := toolbarRect(size.X, size.Y); r.Max.X != size.X-20 || r.Dx() != 280 {
			t.Fatal("right toolbar does not contain three slots")
		}
		for i := 0; i < 3; i++ {
			r := toolbarSlot(size.X, size.Y, i)
			if shown.GrayAt(r.Min.X, r.Min.Y).Y != 68 || shown.GrayAt(r.Max.X-1, r.Max.Y-1).Y != 68 {
				t.Fatal("rounded corner overwrote content")
			}
		}
	}
}

func TestSleepHasNoOverlaysEvenWithLegacyHideSetting(t *testing.T) {
	for _, size := range []image.Point{{1072, 1448}, {1448, 1072}} {
		raw := image.NewGray(image.Rect(0, 0, size.X, size.Y))
		for i := range raw.Pix {
			raw.Pix[i] = byte((i % 16) * 17)
		}
		for _, legacyHide := range []bool{false, true} {
			asleep := composeToolbar(raw, true, true, legacyHide, deviceStatus{Wifi: true, WifiBars: 3, Battery: 89})
			if !bytes.Equal(raw.Pix, asleep.Pix) {
				t.Fatal("sleep contains QR, icons, or a sleep label")
			}
		}
	}
}

func TestCloseWorksAndSleepRestoresSavedPixels(t *testing.T) {
	b := testBoard(t)
	if w := request(b, "POST", "/api/frame", b.token, frame(t, 1072, 1448)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(b, "POST", "/api/settings", b.token, []byte(`{"autoHide":true}`)); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if b.autoHide || !b.barVisible {
		t.Fatal("legacy settings hid awake controls")
	}
	canonical, _ := os.ReadFile(filepath.Join(b.dir, "content.png"))
	asleep, _ := os.ReadFile(filepath.Join(b.dir, "sleep", "board.png"))
	if !bytes.Equal(canonical, asleep) {
		t.Fatal("sleep is not the unmodified posted image")
	}
	closed := false
	b.shutdown = func() { closed = true }
	b.toolbarTap(3)
	if !b.barVisible || closed {
		t.Fatal("content tap hid controls or closed")
	}
	b.toolbarTap(2)
	if !closed {
		t.Fatal("close control did not shut down")
	}
}

func TestQRControlsRemainAvailable(t *testing.T) {
	b := testBoard(t)
	b.autoHide = true
	b.url = "http://192.168.1.2:8080/#test-token"
	if err := b.makeAndShowQR(); err != nil {
		t.Fatal(err)
	}
	if !b.showingQR || !b.barVisible {
		t.Fatal("QR screen hid its controls")
	}
	if w := request(b, "POST", "/api/settings", b.token, []byte(`{}`)); w.Code != 400 {
		t.Fatal("invalid legacy settings accepted")
	}
}

func TestInlineQRCacheAndPrivateEndpoint(t *testing.T) {
	b := testBoard(t)
	b.url = "http://192.0.2.10:8080/#" + strings.Repeat("a", 48)
	w := request(b, "GET", "/api/connection-qr", b.token, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	img, err := png.Decode(bytes.NewReader(w.Body.Bytes()))
	if err != nil || img.Bounds() != image.Rect(0, 0, 192, 192) {
		t.Fatal("small QR dimensions", err)
	}
	modules := len(b.qrCode.Bitmap())
	if 192/modules < 4 {
		t.Fatal("normal connection URL has fewer than four pixels per module")
	}
	before := append([]byte(nil), b.smallQRPNG...)
	b.url = "http://192.0.2.11:8080/#" + strings.Repeat("a", 48)
	if err := b.ensureConnectionQR(); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, b.smallQRPNG) {
		t.Fatal("address change left a stale QR")
	}
}
