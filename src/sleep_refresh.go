package main

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
)

// Explicit black and white X11 frames clear high-contrast QR remnants before
// returning to the photo. Refreshing the existing photo alone left faint lines
// on the tested PW4. All frames still go through X11 to keep its backing pixels
// synchronized; FBInk only requests and waits for a flashing GC16 update.
func (b *board) prepareSleepRefresh() error {
	for name, ink := range map[string]color.Gray{"refresh-black.png": {Y: 0}, "refresh-white.png": {Y: 255}} {
		img := image.NewGray(image.Rect(0, 0, b.width, b.height))
		draw.Draw(img, img.Bounds(), image.NewUniform(ink), image.Point{}, draw.Src)
		if err := writePNG(filepath.Join(b.dir, name), img); err != nil {
			return err
		}
	}
	return nil
}

func (b *board) refreshSleepDisplay(refresh func(string) error) error {
	var result error
	for _, path := range []string{filepath.Join(b.dir, "refresh-black.png"), filepath.Join(b.dir, "refresh-white.png"), filepath.Join(b.dir, "sleep", "board.png")} {
		// Always attempt the posted image, including if an intermediate refresh fails.
		result = errors.Join(result, refresh(path))
	}
	return result
}
