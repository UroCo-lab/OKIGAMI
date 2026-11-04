package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

const connectionQRSize = 192

func connectionQRRect(w, h int) image.Rectangle {
	return image.Rect(20, h-connectionQRSize-20, 20+connectionQRSize, h-20)
}

func (b *board) ensureConnectionQR() error {
	url := b.url
	if url == "" {
		url = "http://127.0.0.1:8080/#" + b.token
	}
	if b.hook != "" {
		port := "8080"
		if parts := strings.Split(strings.Split(url, "/#")[0], ":"); len(parts) > 2 {
			port = parts[len(parts)-1]
		}
		if ip := localIP(); ip != "127.0.0.1" {
			url = "http://" + net.JoinHostPort(ip, port) + "/#" + b.token
		}
	}
	if b.smallQR != nil && b.qrURL == url {
		return nil
	}
	code, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return err
	}
	// Keep modules an integer number of pixels, with the library's quiet zone.
	modules := len(code.Bitmap())
	modulePixels := connectionQRSize / modules
	if modulePixels < 1 {
		modulePixels = 1
	}
	encoded := code.Image(modules * modulePixels)
	qr := image.NewGray(image.Rect(0, 0, connectionQRSize, connectionQRSize))
	draw.Draw(qr, qr.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	at := image.Pt((connectionQRSize-encoded.Bounds().Dx())/2, (connectionQRSize-encoded.Bounds().Dy())/2)
	draw.Draw(qr, image.Rectangle{Min: at, Max: at.Add(encoded.Bounds().Size())}, encoded, encoded.Bounds().Min, draw.Src)
	var data bytes.Buffer
	if err = png.Encode(&data, qr); err != nil {
		return err
	}
	b.url = url
	b.qrURL = url
	b.qrCode = code
	b.smallQR = qr
	b.smallQRPNG = data.Bytes()
	return nil
}
