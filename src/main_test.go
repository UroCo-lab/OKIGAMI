package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func testBoard(t *testing.T) *board {
	t.Helper()
	return &board{token: "test-token", dir: t.TempDir(), width: 1072, height: 1448, shutdown: func() {}}
}
func request(b *board, method, path, token string, data []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	r.Header.Set("X-Board-Token", token)
	w := httptest.NewRecorder()
	b.handler().ServeHTTP(w, r)
	return w
}
func frame(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.SetNRGBA(1, 1, color.NRGBA{R: 80, G: 80, B: 80, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestPrivateEndpointsRequireToken(t *testing.T) {
	b := testBoard(t)
	for _, path := range []string{"/api/config", "/api/current", "/api/connection-qr", "/api/frame", "/api/qr", "/api/sleep", "/api/stop", "/api/settings", "/api/activity"} {
		if got := request(b, "POST", path, "", nil).Code; got != http.StatusUnauthorized {
			t.Fatalf("%s: %d", path, got)
		}
	}
}
func TestUploadRejectsBadAndWrongSizedImages(t *testing.T) {
	b := testBoard(t)
	for _, data := range [][]byte{[]byte("not PNG"), frame(t, 200, 200)} {
		if got := request(b, "POST", "/api/frame", b.token, data).Code; got != 400 {
			t.Fatalf("got %d", got)
		}
	}
	if _, err := os.Stat(filepath.Join(b.dir, "current.png")); !os.IsNotExist(err) {
		t.Fatal("bad frame replaced current")
	}
}
func TestFrameCompositesWhiteAndQuantizes(t *testing.T) {
	b := testBoard(t)
	w := request(b, "POST", "/api/frame", b.token, frame(t, 1072, 1448))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(b.dir, "current.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := color.GrayModel.Convert(img.At(0, 0)).(color.Gray).Y; got != 255 {
		t.Fatalf("transparent background: %d", got)
	}
	if got := color.GrayModel.Convert(img.At(1, 1)).(color.Gray).Y; got != 85 {
		t.Fatalf("gray value: %d", got)
	}
	if got := request(b, "GET", "/api/current", b.token, nil).Code; got != 200 {
		t.Fatal(got)
	}
}
func TestSleepRequiresSavedFrame(t *testing.T) {
	b := testBoard(t)
	if got := request(b, "POST", "/api/sleep", b.token, nil).Code; got != 409 {
		t.Fatal(got)
	}
	if _, err := os.Stat(filepath.Join(b.dir, "sleep-request")); !os.IsNotExist(err) {
		t.Fatal("empty session scheduled sleep")
	}
}
func TestQRHasPanelDimensions(t *testing.T) {
	b := testBoard(t)
	b.url = "http://192.168.1.2:8080/#test-token"
	if err := b.makeQR(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(b.dir, "qr.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil || cfg.Width != 1072 || cfg.Height != 1448 {
		t.Fatal(cfg, err)
	}
}
func TestUploadCannotWriteArbitraryPath(t *testing.T) {
	b := testBoard(t)
	if got := request(b, "POST", "/../../other.png", b.token, frame(t, 1072, 1448)).Code; got != 404 {
		t.Fatal(got)
	}
}

func TestDisplayFailurePreservesSavedFrame(t *testing.T) {
	b := testBoard(t)
	old := []byte("previous saved frame")
	path := filepath.Join(b.dir, "current.png")
	if err := os.WriteFile(path, old, 0600); err != nil {
		t.Fatal(err)
	}
	b.hook = filepath.Join(b.dir, "missing-display-hook.sh")
	if got := request(b, "POST", "/api/frame", b.token, frame(t, 1072, 1448)).Code; got != 500 {
		t.Fatal(got)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, old) {
		t.Fatal("saved frame changed after failed drawing", err)
	}
}

func TestUploadSizeLimit(t *testing.T) {
	b := testBoard(t)
	if got := request(b, "POST", "/api/frame", b.token, make([]byte, (6<<20)+1)).Code; got != 413 {
		t.Fatal(got)
	}
}
