package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadingFrameKeepsSavedContentAndOrientation(t *testing.T) {
	for _, landscape := range []bool{false, true} {
		b := testBoard(t)
		b.landscape = landscape
		saved := []byte("existing user content must stay intact")
		contentPath := filepath.Join(b.dir, "content.png")
		if err := os.WriteFile(contentPath, saved, 0600); err != nil {
			t.Fatal(err)
		}
		path, err := b.writeLoadingFrame()
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds() != image.Rect(0, 0, 1072, 1448) {
			t.Fatal("loading image does not match physical panel")
		}
		logical := rotateToLogical(img, landscape)
		dark := 0
		for _, v := range logical.Pix {
			if v%17 != 0 {
				t.Fatal("loading uses unsupported gray level")
			}
			if v < 153 {
				dark++
			}
		}
		if dark < 1000 {
			t.Fatal("loading image is blank or label missing")
		}
		after, err := os.ReadFile(contentPath)
		if err != nil || !bytes.Equal(saved, after) {
			t.Fatal("loading overwrote saved content")
		}
		if output := os.Getenv("KINDLE_BOARD_VISUAL_OUTPUT"); output != "" {
			name := "loading-portrait.png"
			if landscape {
				name = "loading-landscape.png"
			}
			if err := writePNG(filepath.Join(output, name), logical); err != nil {
				t.Fatal(err)
			}
		}
	}
}
