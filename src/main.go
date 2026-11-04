package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

//go:embed web/index.html
var editor []byte

type board struct {
	mu                             sync.Mutex
	token, dir, hook, url          string
	listenPort                     string
	lookupIP                       func() string
	width, height                  int
	shutdown                       func()
	landscape, showingQR, sleeping bool
	loading                        bool
	nativePaint                    func(string) error
	nativeShow                     func(bool)
	content                        *image.Gray
	autoHide, barVisible, closing  bool
	barEpoch                       uint64
	qrURL                          string
	qrCode                         *qrcode.QRCode
	smallQR                        *image.Gray
	smallQRPNG                     []byte
	sleepAfterSeconds              int
	lastActivity, lastPowerTouch   time.Time
	sleepPending                   bool
}

func (b *board) action(action, path string) error {
	if b.hook == "" {
		return nil
	}
	if (action == "display" || action == "qr" || action == "sleep-refresh") && b.nativePaint != nil {
		if err := b.nativePaint(path); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/bin/sh", b.hook, action, path).CombinedOutput()
	if err != nil {
		log.Printf("display action %s failed: %s", action, output)
		return fmt.Errorf("Kindle display action failed: %w", err)
	}
	return nil
}

func (b *board) authorized(r *http.Request) bool {
	supplied := r.Header.Get("X-Board-Token")
	return len(supplied) == len(b.token) && subtle.ConstantTimeCompare([]byte(supplied), []byte(b.token)) == 1
}

func (b *board) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' blob: data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'")
		if serveEditorFont(w, r) {
			return
		}
		if r.URL.Path == "/" && r.Method == "GET" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(editor)
			return
		}
		if !b.authorized(r) {
			http.Error(w, "接続用QRコードを読み直してください。", http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/api/config" && r.Method == "GET":
			b.mu.Lock()
			defer b.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"width": b.width, "height": b.height, "version": "0.6.1", "device": "Kindle Paperwhite 4", "sleeping": b.sleeping, "landscape": b.landscape, "preview": b.hook == "", "autoHide": b.autoHide, "barVisible": b.barVisible, "sleepAfterSeconds": b.sleepAfterSeconds, "status": readDeviceStatus()})
		case r.URL.Path == "/api/current" && r.Method == "GET":
			b.mu.Lock()
			defer b.mu.Unlock()
			data, err := os.ReadFile(filepath.Join(b.dir, "current.png"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(data)
		case r.URL.Path == "/api/connection-qr" && r.Method == "GET":
			b.mu.Lock()
			defer b.mu.Unlock()
			if err := b.ensureConnectionQR(); err != nil {
				http.Error(w, "QRコードを準備できませんでした。", 500)
				return
			}
			if b.smallQR == nil {
				http.Error(w, "KindleをWi-Fiへ接続してください。", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(b.smallQRPNG)
		case r.URL.Path == "/api/settings" && r.Method == "POST":
			b.mu.Lock()
			defer b.mu.Unlock()
			var settings struct {
				AutoHide          *bool `json:"autoHide"`
				SleepAfterSeconds *int  `json:"sleepAfterSeconds"`
			}
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&settings); err != nil || (settings.AutoHide == nil && settings.SleepAfterSeconds == nil) {
				http.Error(w, "設定が不正です。", 400)
				return
			}
			if settings.SleepAfterSeconds != nil {
				if !validSleepDelay(*settings.SleepAfterSeconds) {
					http.Error(w, "スリープ時間が不正です。", 400)
					return
				}
				previous := b.sleepAfterSeconds
				b.sleepAfterSeconds = *settings.SleepAfterSeconds
				if err := b.saveSleepSetting(); err != nil {
					b.sleepAfterSeconds = previous
					http.Error(w, "設定を保存できませんでした。", 500)
					return
				}
				b.noteActivityLocked(time.Now())
			}
			// Old clients may still submit this setting; awake overlays stay visible.
			b.autoHide = false
			b.barEpoch++
			if !b.showingQR {
				b.barVisible = true
			}
			if err := b.saveToolbarSetting(); err != nil {
				http.Error(w, "設定を保存できませんでした。", 500)
				return
			}
			if settings.AutoHide != nil && !b.sleeping && !b.showingQR {
				if err := b.renderContent(); err != nil {
					http.Error(w, "設定は保存されましたが表示を更新できませんでした。", 500)
					return
				}
			}
			w.WriteHeader(204)
		case r.URL.Path == "/api/activity" && r.Method == "POST":
			b.mu.Lock()
			defer b.mu.Unlock()
			if b.sleeping || b.closing {
				http.Error(w, "Kindleを起こしてから操作してください。", 409)
				return
			}
			b.noteActivityLocked(time.Now())
			w.WriteHeader(204)
		case r.URL.Path == "/api/frame" && r.Method == "POST":
			b.upload(w, r)
		case r.URL.Path == "/api/qr" && r.Method == "POST":
			b.mu.Lock()
			defer b.mu.Unlock()
			if b.sleeping {
				http.Error(w, "Kindleを起こしてから操作してください。", 409)
				return
			}
			if err := b.makeAndShowQR(); err != nil {
				http.Error(w, "QRコードを表示できませんでした。", 500)
				return
			}
			w.WriteHeader(204)
		case r.URL.Path == "/api/sleep" && r.Method == "POST":
			b.mu.Lock()
			defer b.mu.Unlock()
			if _, err := os.Stat(filepath.Join(b.dir, "current.png")); err != nil {
				http.Error(w, "先に画像または文字を送信してください。", 409)
				return
			}
			if b.sleeping {
				http.Error(w, "すでにスリープ中です。", 409)
				return
			}
			w.WriteHeader(204)
			go func() {
				time.Sleep(500 * time.Millisecond)
				if err := b.action("sleep", filepath.Join(b.dir, "current.png")); err != nil {
					b.reportError("request sleep", err)
				}
			}()
		case r.URL.Path == "/api/stop" && r.Method == "POST":
			b.mu.Lock()
			defer b.mu.Unlock()
			_ = os.Remove(filepath.Join(b.dir, "sleep-request"))
			w.WriteHeader(204)
			go func() { time.Sleep(500 * time.Millisecond); b.shutdown() }()
		default:
			http.NotFound(w, r)
		}
	})
}

func (b *board) upload(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sleeping {
		http.Error(w, "Kindleを起こしてから送信してください。", 409)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "画像が大きすぎるか、受信できませんでした。", 413)
		return
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	portrait := cfg.Width == b.width && cfg.Height == b.height
	landscape := cfg.Width == b.height && cfg.Height == b.width
	if err != nil || (!portrait && !landscape) {
		http.Error(w, "画面サイズに合うPNGが必要です。", 400)
		return
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "PNGを読み込めませんでした。", 400)
		return
	}
	// Composite transparency on white, then quantize to the panel's 16 gray levels.
	flat := image.NewRGBA(img.Bounds())
	draw.Draw(flat, flat.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, image.Point{}, draw.Over)
	gray := image.NewGray(img.Bounds())
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			v := color.GrayModel.Convert(flat.At(x, y)).(color.Gray).Y
			gray.SetGray(x, y, color.Gray{Y: byte((int(v) + 8) / 17 * 17)})
		}
	}
	oldContent, oldLandscape, oldAutoHide, oldBar := b.content, b.landscape, b.autoHide, b.barVisible
	switch r.Header.Get("X-Board-Auto-Hide") {
	case "1", "0", "":
	default:
		http.Error(w, "操作バーの設定が不正です。", 400)
		return
	}
	b.content = gray
	b.landscape = landscape
	b.autoHide = false
	b.barVisible = true
	b.showingQR = false
	b.barEpoch++
	if err = b.renderContent(); err != nil {
		b.content = oldContent
		b.landscape = oldLandscape
		b.autoHide = oldAutoHide
		b.barVisible = oldBar
		http.Error(w, "Kindleに表示できませんでした。前の保存画像は保持しています。", 500)
		return
	}
	rawPath := filepath.Join(b.dir, "content.tmp.png")
	if err = writePNG(rawPath, rotateToPortrait(gray, landscape)); err == nil {
		err = replaceFile(rawPath, filepath.Join(b.dir, "content.png"))
	}
	if err != nil {
		http.Error(w, "表示は更新されましたが元画像を保存できませんでした。", 500)
		return
	}
	if err = b.saveToolbarSetting(); err != nil {
		b.reportError("toolbar settings", err)
	}
	orientation := []byte("portrait")
	if landscape {
		orientation = []byte("landscape")
	}
	if err = os.WriteFile(filepath.Join(b.dir, "orientation.txt"), orientation, 0600); err != nil {
		b.reportError("save orientation", err)
	}
	b.noteActivityLocked(time.Now())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "updated": time.Now().Format(time.RFC3339)})
}

func replaceFile(src, dst string) error {
	// Rename overwrites atomically on Kindle/Linux. Windows desktop preview needs removal.
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrExist) || strings.Contains(err.Error(), "already exists") {
		if e := os.Remove(dst); e != nil {
			return e
		}
		return os.Rename(src, dst)
	}
	return err
}

func localIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err == nil && ip.To4() != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}
	return "127.0.0.1"
}

func (b *board) makeQR() error {
	if err := b.ensureConnectionQR(); err != nil {
		return err
	}
	w, h := b.width, b.height
	if b.landscape {
		w, h = b.height, b.width
	}
	img := image.NewGray(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	if b.qrCode != nil {
		size := min(w, h) * 3 / 4
		qr := b.qrCode.Image(size)
		at := image.Pt((w-size)/2, (h-size-100)/2)
		draw.Draw(img, image.Rectangle{Min: at, Max: at.Add(qr.Bounds().Size())}, qr, image.Point{}, draw.Src)
	} else {
		label, err := png.Decode(bytes.NewReader(offlineLabel))
		if err != nil {
			return err
		}
		at := image.Pt((w-label.Bounds().Dx())/2, (h-label.Bounds().Dy())/2)
		draw.Draw(img, image.Rectangle{Min: at, Max: at.Add(label.Bounds().Size())}, label, label.Bounds().Min, draw.Src)
	}
	f, err := os.Create(filepath.Join(b.dir, "qr.png"))
	if err != nil {
		return err
	}
	err = png.Encode(f, rotateToPortrait(b.compose(img, false), b.landscape))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if _, e := os.Stat(filepath.Join(b.dir, "current.png")); os.IsNotExist(e) {
		blank := image.NewGray(img.Bounds())
		draw.Draw(blank, blank.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		return b.writeSleepFrame(blank)
	}
	return nil
}

func main() {
	addr := flag.String("listen", ":8080", "listen address")
	dir := flag.String("data", "state", "data directory")
	hook := flag.String("hook", "", "display hook, empty for desktop preview")
	publicIP := flag.String("ip", "", "IP encoded in QR")
	width := flag.Int("width", 1072, "portrait width")
	height := flag.Int("height", 1448, "portrait height")
	lifetime := flag.Duration("timeout", 0, "optional receiving session limit; 0 keeps running across normal suspend")
	selftest := flag.Bool("selftest", false, "write a QR without opening a listener or drawing")
	flag.Parse()
	if *width < 100 || *height < 100 || *width > 2048 || *height > 2048 {
		log.Fatal("invalid resolution")
	}
	absolute, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.MkdirAll(absolute, 0700); err != nil {
		log.Fatal(err)
	}
	tokenBytes := make([]byte, 24)
	if _, err = rand.Read(tokenBytes); err != nil {
		log.Fatal(err)
	}
	token := hex.EncodeToString(tokenBytes)
	if *publicIP == "" {
		*publicIP = localIP()
	}
	_, port, err := net.SplitHostPort(*addr)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = strconv.Atoi(port); err != nil {
		log.Fatal(err)
	}
	b := &board{barVisible: true, token: token, dir: absolute, hook: *hook, listenPort: port, width: *width, height: *height, url: "http://" + net.JoinHostPort(*publicIP, port) + "/#" + token}
	if orientation, e := os.ReadFile(filepath.Join(absolute, "orientation.txt")); e == nil {
		b.landscape = strings.TrimSpace(string(orientation)) == "landscape"
	}
	b.loadSleepSetting()
	server := &http.Server{Addr: *addr, Handler: b.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.shutdown = func() { b.mu.Lock(); b.closing = true; b.barEpoch++; b.mu.Unlock(); cancel() }
	_ = os.Remove(filepath.Join(absolute, "sleep-request"))
	if *selftest {
		if err = b.makeQR(); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("SELFTEST_OK: runtime and PNG/QR encoder (%dx%d)\n", *width, *height)
		return
	}
	// Show the startup screen before QR generation and stored-image preparation.
	b.loading = true
	b.showingQR = true
	b.barVisible = true
	loadingPath, err := b.writeLoadingFrame()
	if err != nil {
		log.Fatal(err)
	}
	closeNative, err := b.startNative(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer closeNative()
	if err = b.action("display", loadingPath); err != nil {
		log.Fatal(err)
	}
	if err = b.makeQR(); err != nil {
		log.Fatal(err)
	}
	if err = b.prepareStoredFrame(); err != nil {
		log.Fatal(err)
	}
	if err = b.prepareSleepRefresh(); err != nil {
		log.Fatal(err)
	}
	b.barVisible = true
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	initialAction, initialPath := "qr", filepath.Join(absolute, "qr.png")
	if b.content != nil {
		b.showingQR = false
		initialAction, initialPath = "display", filepath.Join(absolute, "current.png")
	}
	if err = b.action(initialAction, initialPath); err != nil {
		listener.Close()
		log.Fatal(err)
	}
	activeMarker := filepath.Join(absolute, "receiver-active")
	if err = os.WriteFile(activeMarker, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		log.Fatal(err)
	}
	b.mu.Lock()
	b.loading = false
	b.lastActivity = time.Now()
	b.mu.Unlock()
	b.watchPower(ctx)
	b.watchStatus(ctx)
	b.watchAutoSleep(ctx)
	log.Printf("OKIGAMI 0.6.1 listening on %s; display ready (%dx%d)", *addr, *width, *height)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	go func() {
		select {
		case <-signals:
			b.shutdown()
		case <-ctx.Done():
		}
	}()
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
			cancel()
		}
	}()
	var timeout <-chan time.Time
	if *lifetime > 0 {
		timer := time.NewTimer(*lifetime)
		defer timer.Stop()
		timeout = timer.C
	}
	select {
	case <-ctx.Done():
	case <-timeout:
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}
