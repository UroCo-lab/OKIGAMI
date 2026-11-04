package main

import "testing"

func TestEditorFontsAreLocalImmutableAssets(t *testing.T) {
	b := testBoard(t)
	for _, path := range []string{"/fonts/v1/gothic.ttf", "/fonts/v1/mincho.ttf", "/fonts/v1/gothic-bold.ttf", "/fonts/v1/mincho-bold.ttf", "/fonts/v1/rounded.ttf", "/fonts/v1/rounded-bold.ttf"} {
		w := request(b, "GET", path, "", nil)
		if w.Code != 200 || w.Header().Get("Content-Type") != "font/ttf" || w.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
			t.Fatal(path, w.Code, w.Header())
		}
		data := w.Body.Bytes()
		if len(data) < 1000000 || string(data[:4]) != "\x00\x01\x00\x00" {
			t.Fatal("invalid TrueType font", path)
		}
	}
	if w := request(b, "GET", "/fonts/v1/../../connection.txt", "", nil); w.Code != 401 {
		t.Fatal("font route exposed private state", w.Code)
	}
}
