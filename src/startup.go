package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"path/filepath"
)

// Rasterized Japanese UI text keeps startup independent of device fonts.
//
//go:embed assets/loading-label.png
var loadingLabel []byte

func (b *board) writeLoadingFrame() (string, error) {
	label, err := png.Decode(bytes.NewReader(loadingLabel))
	if err != nil {
		return "", err
	}
	w, h := b.width, b.height
	if b.landscape {
		w, h = h, w
	}
	img := image.NewGray(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	at := image.Pt((w-label.Bounds().Dx())/2, (h-label.Bounds().Dy())/2+35)
	draw.Draw(img, image.Rectangle{Min: at, Max: at.Add(label.Bounds().Size())}, label, label.Bounds().Min, draw.Src)
	// A static hourglass avoids repeated e-ink flashes while preparing the QR.
	cx, y := w/2, h/2-180
	line(img, cx-25, y, cx+25, y, 4, 0)
	line(img, cx-25, y+60, cx+25, y+60, 4, 0)
	line(img, cx-21, y+4, cx+21, y+56, 3, 0)
	line(img, cx+21, y+4, cx-21, y+56, 3, 0)
	path := filepath.Join(b.dir, "loading.png")
	return path, writePNG(path, rotateToPortrait(img, b.landscape))
}
