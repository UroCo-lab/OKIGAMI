package main

import (
	"image"
	"image/color"
	"testing"
)

func TestX11GrayPackingAndScanlinePadding(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 3, 1))
	img.SetGray(0, 0, color.Gray{Y: 0})
	img.SetGray(1, 0, color.Gray{Y: 85})
	img.SetGray(2, 0, color.Gray{Y: 255})
	data, stride, err := packXPixels(img, 8, 32, 0, 255, 0, 0, 0, 0)
	if err != nil || stride != 4 || len(data) != 4 || data[0] != 0 || data[1] != 85 || data[2] != 255 {
		t.Fatal(data, stride, err)
	}
	inverted, _, err := packXPixels(img, 8, 32, 0, 0, 255, 0, 0, 0)
	if err != nil || inverted[0] != 255 || inverted[2] != 0 {
		t.Fatal(inverted, err)
	}
}

func TestX11RGB565ByteOrder(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	img.SetGray(0, 0, color.Gray{Y: 255})
	data, stride, err := packXPixels(img, 16, 16, 0, 0xffff, 0, 0xf800, 0x07e0, 0x001f)
	if err != nil || stride != 2 || data[0] != 255 || data[1] != 255 {
		t.Fatal(data, stride, err)
	}
	img.SetGray(0, 0, color.Gray{Y: 17})
	le, _, _ := packXPixels(img, 16, 16, 0, 0xffff, 0, 0xf800, 0x07e0, 0x001f)
	be, _, _ := packXPixels(img, 16, 16, 1, 0xffff, 0, 0xf800, 0x07e0, 0x001f)
	if le[0] != be[1] || le[1] != be[0] {
		t.Fatal(le, be)
	}
}

func TestX11UnsupportedFormatsFailBeforePainting(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	if _, _, err := packXPixels(img, 4, 32, 0, 15, 0, 0, 0, 0); err == nil {
		t.Fatal("accepted unsupported depth")
	}
}
