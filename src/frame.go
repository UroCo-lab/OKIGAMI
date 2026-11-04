package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

func rotateToPortrait(src image.Image, landscape bool) *image.Gray {
	if !landscape {
		dst := image.NewGray(src.Bounds())
		draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
		return dst
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewGray(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.SetGray(h-1-y, x, color.GrayModel.Convert(src.At(x, y)).(color.Gray))
		}
	}
	return dst
}

func rotateToLogical(src image.Image, landscape bool) *image.Gray {
	if !landscape {
		return rotateToPortrait(src, false)
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewGray(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.SetGray(y, w-1-x, color.GrayModel.Convert(src.At(x, y)).(color.Gray))
		}
	}
	return dst
}

func writePNG(path string, img image.Image) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func (b *board) compose(src image.Image, sleeping bool) *image.Gray {
	return composeToolbar(src, sleeping, true, false, readDeviceStatus(), b.smallQR)
}
func (b *board) writeSleepFrame(logical image.Image) error {
	if err := os.MkdirAll(filepath.Join(b.dir, "sleep"), 0700); err != nil {
		return err
	}
	tmp := filepath.Join(b.dir, "sleep", "board.tmp.png")
	if err := writePNG(tmp, rotateToPortrait(b.compose(logical, true), b.landscape)); err != nil {
		return err
	}
	return replaceFile(tmp, filepath.Join(b.dir, "sleep", "board.png"))
}
func (b *board) renderContent() error {
	if b.closing {
		return nil
	}
	if b.content == nil {
		return b.makeAndShowQR()
	}
	if err := b.ensureConnectionQR(); err != nil {
		return err
	}
	tmp := filepath.Join(b.dir, "candidate.png")
	if err := writePNG(tmp, rotateToPortrait(b.compose(b.content, false), b.landscape)); err != nil {
		return err
	}
	if err := b.action("display", tmp); err != nil {
		return err
	}
	if err := replaceFile(tmp, filepath.Join(b.dir, "current.png")); err != nil {
		return err
	}
	return b.writeSleepFrame(b.content)
}
func (b *board) showCurrent() error {
	if b.closing {
		return nil
	}
	if b.content == nil {
		return b.makeAndShowQR()
	}
	b.showingQR = false
	b.barVisible = true
	return b.renderContent()
}
func (b *board) makeAndShowQR() error {
	if b.closing {
		return nil
	}
	b.showingQR = true
	b.barVisible = true
	b.barEpoch++
	if err := b.makeQR(); err != nil {
		return err
	}
	return b.action("qr", filepath.Join(b.dir, "qr.png"))
}
func (b *board) toggleQR() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sleeping || b.closing {
		return
	}
	var err error
	if b.showingQR {
		err = b.showCurrent()
	} else {
		err = b.makeAndShowQR()
	}
	if err != nil {
		b.reportError("toggle QR", err)
	}
}
func (b *board) prepareStoredFrame() error {
	path := filepath.Join(b.dir, "content.png")
	data, err := os.ReadFile(path)
	legacy := os.IsNotExist(err)
	if legacy {
		data, err = os.ReadFile(filepath.Join(b.dir, "current.png"))
	}
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	logical := rotateToLogical(img, b.landscape)
	if legacy {
		w, h := logical.Bounds().Dx(), logical.Bounds().Dy()
		draw.Draw(logical, image.Rect(w-240, h-116, w-20, h-20), image.NewUniform(color.White), image.Point{}, draw.Src)
		if err = writePNG(path, rotateToPortrait(logical, b.landscape)); err != nil {
			return err
		}
	}
	b.content = logical
	b.barVisible = true
	if err = writePNG(filepath.Join(b.dir, "current.png"), rotateToPortrait(b.compose(logical, false), b.landscape)); err != nil {
		return err
	}
	return b.writeSleepFrame(logical)
}
