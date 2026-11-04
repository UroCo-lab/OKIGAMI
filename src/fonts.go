package main

import (
	"embed"
	"net/http"
)

// Fonts are served by the Kindle itself; editing never contacts a font CDN.
//
//go:embed web/fonts/*.ttf
var editorFonts embed.FS

func serveEditorFont(w http.ResponseWriter, r *http.Request) bool {
	fonts := map[string]string{
		"/fonts/v1/gothic.ttf":       "web/fonts/bizudgothic-BIZUDGothic-Regular.ttf",
		"/fonts/v1/mincho.ttf":       "web/fonts/bizudmincho-BIZUDMincho-Regular.ttf",
		"/fonts/v1/gothic-bold.ttf":  "web/fonts/bizudgothic-BIZUDGothic-Bold.ttf",
		"/fonts/v1/mincho-bold.ttf":  "web/fonts/bizudmincho-BIZUDMincho-Bold.ttf",
		"/fonts/v1/rounded.ttf":      "web/fonts/zenmarugothic-ZenMaruGothic-Regular.ttf",
		"/fonts/v1/rounded-bold.ttf": "web/fonts/zenmarugothic-ZenMaruGothic-Bold.ttf",
	}
	file, ok := fonts[r.URL.Path]
	if !ok || r.Method != "GET" {
		return false
	}
	data, err := editorFonts.ReadFile(file)
	if err != nil {
		http.Error(w, "Font unavailable", 500)
		return true
	}
	w.Header().Set("Content-Type", "font/ttf")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(data)
	return true
}
