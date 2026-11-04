package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestOfflineStartupRestoresSavedContentAndSleepImage(t *testing.T) {
	for _, landscape := range []bool{false, true} {
		b := testBoard(t)
		b.landscape = landscape
		b.lookupIP = func() string { return "127.0.0.1" }
		b.url = "http://192.0.2.1:8080/#old-session"
		w, h := b.width, b.height
		if landscape {
			w, h = h, w
		}
		raw := image.NewGray(image.Rect(0, 0, w, h))
		draw.Draw(raw, raw.Bounds(), image.NewUniform(color.Gray{Y: 85}), image.Point{}, draw.Src)
		raw.SetGray(300, 400, color.Gray{Y: 17})
		contentPath := filepath.Join(b.dir, "content.png")
		if err := writePNG(contentPath, rotateToPortrait(raw, landscape)); err != nil {
			t.Fatal(err)
		}
		before, _ := os.ReadFile(contentPath)
		if err := b.makeQR(); err != nil {
			t.Fatal(err)
		}
		if err := b.prepareStoredFrame(); err != nil {
			t.Fatal(err)
		}
		if b.content == nil || b.smallQR != nil || b.qrCode != nil || b.url != "" {
			t.Fatal("offline startup lost content or retained an unusable QR")
		}
		after, _ := os.ReadFile(contentPath)
		if !bytes.Equal(before, after) {
			t.Fatal("saved original was modified")
		}
		for _, name := range []string{"current.png", "sleep/board.png"} {
			data, err := os.ReadFile(filepath.Join(b.dir, name))
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			logical := rotateToLogical(decoded, landscape)
			if !bytes.Equal(logical.Pix[:w*(h-130)], raw.Pix[:w*(h-130)]) {
				t.Fatalf("%s changed saved pixels above the controls", name)
			}
			qrArea := connectionQRRect(w, h)
			for y := qrArea.Min.Y; y < qrArea.Max.Y; y++ {
				for x := qrArea.Min.X; x < qrArea.Max.X; x++ {
					if logical.GrayAt(x, y) != raw.GrayAt(x, y) {
						t.Fatal("offline frame contains a stale QR")
					}
				}
			}
			if name == "sleep/board.png" && !bytes.Equal(logical.Pix, raw.Pix) {
				t.Fatal("sleep content differs")
			}
			if output := os.Getenv("KINDLE_BOARD_VISUAL_OUTPUT"); output != "" && name == "current.png" {
				label := "offline-portrait.png"
				if landscape {
					label = "offline-landscape.png"
				}
				if err := writePNG(filepath.Join(output, label), logical); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestConnectionQRTracksOfflineAndReconnectedAddress(t *testing.T) {
	b := testBoard(t)
	ip := "127.0.0.1"
	b.lookupIP = func() string { return ip }
	b.listenPort = "8081"
	b.url = "http://127.0.0.1:8081/#" + b.token
	for _, next := range []string{"127.0.0.1", "192.0.2.10", "127.0.0.1", "192.0.2.11"} {
		ip = next
		if err := b.ensureConnectionQR(); err != nil {
			t.Fatal(err)
		}
		connection, err := os.ReadFile(filepath.Join(b.dir, "connection.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if ip == "127.0.0.1" {
			if b.smallQR != nil || b.url != "" || len(connection) != 0 {
				t.Fatal("offline retained a connection QR")
			}
			if got := request(b, "GET", "/api/connection-qr", b.token, nil).Code; got != 503 {
				t.Fatal(got)
			}
		} else {
			want := "http://" + ip + ":8081/#" + b.token
			if b.smallQR == nil || b.url != want || string(connection) != want+"\n" {
				t.Fatal("QR not updated after reconnect")
			}
		}
	}
}
