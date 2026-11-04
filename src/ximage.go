package main

import (
	"fmt"
	"image"
	"image/color"
	"math/bits"
)

func xChannel(v byte, mask uint32) uint32 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	max := mask >> shift
	return uint32((uint64(v)*uint64(max)+127)/255) << shift
}

// X11's pixel format is independent of the e-ink framebuffer's memory format.
func packXPixels(img image.Image, bpp, pad, byteOrder byte, white, black, red, green, blue uint32) ([]byte, int, error) {
	if bpp != 8 && bpp != 16 && bpp != 24 && bpp != 32 {
		return nil, 0, fmt.Errorf("unsupported X11 bits per pixel: %d", bpp)
	}
	if pad != 8 && pad != 16 && pad != 32 {
		return nil, 0, fmt.Errorf("unsupported X11 scanline pad: %d", pad)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	stride := ((w*int(bpp) + int(pad) - 1) / int(pad)) * (int(pad) / 8)
	data := make([]byte, stride*h)
	pixelBytes := int(bpp) / 8
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := color.GrayModel.Convert(img.At(img.Bounds().Min.X+x, img.Bounds().Min.Y+y)).(color.Gray).Y
			var pixel uint32
			if red|green|blue != 0 {
				pixel = xChannel(v, red) | xChannel(v, green) | xChannel(v, blue)
			} else {
				pixel = uint32(int64(black) + (int64(white)-int64(black))*int64(v)/255)
			}
			offset := y*stride + x*pixelBytes
			for i := 0; i < pixelBytes; i++ {
				shift := i * 8
				if byteOrder != 0 {
					shift = (pixelBytes - 1 - i) * 8
				}
				data[offset+i] = byte(pixel >> shift)
			}
		}
	}
	return data, stride, nil
}
