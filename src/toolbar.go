package main

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type deviceStatus struct {
	Wifi     bool `json:"wifi"`
	WifiBars int  `json:"wifiBars"` // -1: strength unavailable; 0..3: increasing strength
	Battery  int  `json:"battery"`
	Charging bool `json:"charging"`
}

func readDeviceStatus() deviceStatus {
	s := deviceStatus{Battery: -1, WifiBars: -1}
	if iface, e := net.InterfaceByName("wlan0"); e == nil && iface.Flags&net.FlagUp != 0 {
		addresses, _ := iface.Addrs()
		for _, a := range addresses {
			ip, _, e := net.ParseCIDR(a.String())
			if e == nil && ip.To4() != nil {
				s.Wifi = true
			}
		}
	}
	if s.Wifi {
		data, _ := os.ReadFile("/proc/net/wireless")
		s.WifiBars = wirelessBars(string(data))
		if s.WifiBars < 0 {
			// Some Kindle drivers omit /proc statistics but expose WEXT via iwconfig.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			out, err := exec.CommandContext(ctx, "iwconfig", "wlan0").Output()
			cancel()
			if err == nil {
				s.WifiBars = iwconfigBars(string(out))
			}
		}
	}
	paths, _ := filepath.Glob("/sys/class/power_supply/*/capacity")
	for _, path := range paths {
		data, e := os.ReadFile(path)
		if e != nil {
			continue
		}
		n, e := strconv.Atoi(strings.TrimSpace(string(data)))
		if e == nil && n >= 0 && n <= 100 {
			s.Battery = n
			status, _ := os.ReadFile(filepath.Join(filepath.Dir(path), "status"))
			s.Charging = strings.TrimSpace(string(status)) == "Charging"
			break
		}
	}
	return s
}

// RSSI thresholds are presentation choices; unavailable readings never imply full strength.
func signalBars(dbm int) int {
	if dbm >= -60 {
		return 3
	}
	if dbm >= -72 {
		return 2
	}
	if dbm >= -85 {
		return 1
	}
	return 0
}
func wirelessBars(data string) int {
	for _, row := range strings.Split(data, "\n") {
		name, values, ok := strings.Cut(row, ":")
		if !ok || strings.TrimSpace(name) != "wlan0" {
			continue
		}
		fields := strings.Fields(values)
		if len(fields) < 4 {
			return -1
		}
		level, err := strconv.Atoi(strings.TrimSuffix(fields[2], "."))
		// /proc only identifies dBm unambiguously when the level is negative.
		if err == nil && level >= -127 && level < 0 {
			return signalBars(level)
		}
	}
	return -1
}

var wifiDBM = regexp.MustCompile(`Signal level[=:]\s*(-\d+)\s*dBm`)

func iwconfigBars(data string) int {
	m := wifiDBM.FindStringSubmatch(data)
	if len(m) != 2 {
		return -1
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < -127 || n >= 0 {
		return -1
	}
	return signalBars(n)
}

func toolbarRect(w, h int) image.Rectangle { return image.Rect(w-300, h-108, w-20, h-20) }
func toolbarSlot(w, h, index int) image.Rectangle {
	x := w - 300 + index*96
	return image.Rect(x, h-108, x+88, h-20)
}
func controlRect(w, h int) image.Rectangle { return connectionQRRect(w, h) }
func exitRect(w, h int) image.Rectangle    { return toolbarSlot(w, h, 2) }

func line(img *image.Gray, x0, y0, x1, y1, thickness int, ink byte) {
	dx, dy := x1-x0, y1-y0
	steps := max(abs(dx), abs(dy))
	if steps == 0 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		x := x0 + dx*i/steps
		y := y0 + dy*i/steps
		draw.Draw(img, image.Rect(x-thickness/2, y-thickness/2, x+(thickness+1)/2, y+(thickness+1)/2), image.NewUniform(color.Gray{Y: ink}), image.Point{}, draw.Src)
	}
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func roundedRect(img *image.Gray, r image.Rectangle, radius int, ink byte) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dx := max(r.Min.X+radius-x, 0, x-(r.Max.X-1-radius))
			dy := max(r.Min.Y+radius-y, 0, y-(r.Max.Y-1-radius))
			if dx*dx+dy*dy <= radius*radius {
				img.SetGray(x, y, color.Gray{Y: ink})
			}
		}
	}
}

func drawBattery(img *image.Gray, r image.Rectangle, status deviceStatus, ink byte) {
	body := image.Rect(r.Min.X+9, r.Min.Y+25, r.Min.X+73, r.Min.Y+63)
	roundedRect(img, body, 7, ink)
	inside := body.Inset(3)
	roundedRect(img, inside, 4, 255)
	roundedRect(img, image.Rect(body.Max.X+2, body.Min.Y+12, body.Max.X+7, body.Max.Y-12), 2, ink)
	text := "-"
	if status.Battery >= 0 {
		level := min(status.Battery, 100)
		text = strconv.Itoa(level)
		fill := inside.Inset(2)
		cut := fill.Min.X + fill.Dx()*level/100
		roundedRect(img, fill, 2, ink)
		draw.Draw(img, image.Rect(cut, fill.Min.Y, fill.Max.X, fill.Max.Y), image.NewUniform(color.White), image.Point{}, draw.Src)
	}
	textW := (4*len(text) - 1) * 3
	x, y := body.Min.X+(body.Dx()-textW)/2, body.Min.Y+(body.Dy()-15)/2
	mask := image.NewGray(image.Rect(0, 0, textW, 15))
	draw.Draw(mask, mask.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	tinyText(mask, text, 0, 0, 3, 0)
	for my := 0; my < 15; my++ {
		for mx := 0; mx < textW; mx++ {
			if mask.GrayAt(mx, my).Y == 0 {
				v := ink
				if img.GrayAt(x+mx, y+my).Y == ink {
					v = 255
				}
				img.SetGray(x+mx, y+my, color.Gray{Y: v})
			}
		}
	}
	if status.Charging {
		cx, cy := r.Min.X+44, r.Min.Y+70
		line(img, cx+3, cy-3, cx-4, cy+3, 2, ink)
		line(img, cx-4, cy+3, cx+4, cy+3, 2, ink)
		line(img, cx+4, cy+3, cx-3, cy+9, 2, ink)
	}
}

var tinyFont = map[rune][]string{
	'Z': {"111", "001", "010", "100", "111"},
	'0': {"111", "101", "101", "101", "111"}, '1': {"010", "110", "010", "010", "111"}, '2': {"111", "001", "111", "100", "111"}, '3': {"111", "001", "111", "001", "111"}, '4': {"101", "101", "111", "001", "001"}, '5': {"111", "100", "111", "001", "111"}, '6': {"111", "100", "111", "101", "111"}, '7': {"111", "001", "010", "010", "010"}, '8': {"111", "101", "111", "101", "111"}, '9': {"111", "101", "111", "001", "111"}, '-': {"000", "000", "111", "000", "000"}, '%': {"101", "001", "010", "100", "101"}, 'z': {"000", "111", "010", "100", "111"},
}

func tinyText(img *image.Gray, text string, x, y, scale int, ink byte) {
	for _, r := range text {
		for row, bits := range tinyFont[r] {
			for col, v := range bits {
				if v == '1' {
					draw.Draw(img, image.Rect(x+col*scale, y+row*scale, x+(col+1)*scale, y+(row+1)*scale), image.NewUniform(color.Gray{Y: ink}), image.Point{}, draw.Src)
				}
			}
		}
		x += 4 * scale
	}
}

func composeToolbar(src image.Image, sleeping, visible, autoHide bool, status deviceStatus, qrImages ...image.Image) *image.Gray {
	img := image.NewGray(src.Bounds())
	draw.Draw(img, img.Bounds(), src, src.Bounds().Min, draw.Src)
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if sleeping {
		return img
	}
	if len(qrImages) > 0 && qrImages[0] != nil {
		r := connectionQRRect(w, h)
		draw.Draw(img, r, qrImages[0], qrImages[0].Bounds().Min, draw.Src)
	}
	ink := byte(0)
	for i := 0; i < 3; i++ {
		r := toolbarSlot(w, h, i)
		roundedRect(img, r, 14, 153)
		roundedRect(img, r.Inset(2), 12, 255)
	}
	r := toolbarSlot(w, h, 0)
	cx, cy := r.Min.X+44, r.Min.Y+58
	for y := r.Min.Y + 12; y < r.Min.Y+62; y++ {
		for x := r.Min.X + 12; x < r.Max.X-12; x++ {
			dx, dy := x-cx, y-cy
			d := math.Sqrt(float64(dx*dx + dy*dy))
			for level, radius := range []float64{12, 23, 34} {
				if dy <= 0 && abs(dx) <= -dy+3 && math.Abs(d-radius) < 2 {
					arcInk := byte(204)
					if status.Wifi && status.WifiBars > level {
						arcInk = ink
					}
					img.SetGray(x, y, color.Gray{Y: arcInk})
				}
			}
		}
	}
	draw.Draw(img, image.Rect(cx-3, cy-2, cx+4, cy+5), image.NewUniform(color.Gray{Y: ink}), image.Point{}, draw.Src)
	if !status.Wifi {
		line(img, r.Min.X+18, r.Min.Y+16, r.Max.X-18, r.Max.Y-16, 4, ink)
	}
	r = toolbarSlot(w, h, 1)
	drawBattery(img, r, status, ink)
	r = exitRect(w, h)
	line(img, r.Min.X+25, r.Min.Y+25, r.Max.X-25, r.Max.Y-25, 5, ink)
	line(img, r.Max.X-25, r.Min.Y+25, r.Min.X+25, r.Max.Y-25, 5, ink)
	return img
}

func (b *board) saveToolbarSetting() error {
	data, _ := json.Marshal(map[string]bool{"autoHide": b.autoHide})
	return os.WriteFile(filepath.Join(b.dir, "toolbar.json"), data, 0600)
}
func (b *board) toolbarTap(action int) {
	if action == 2 {
		b.shutdown()
		return
	}
	if action == 1 {
		b.toggleQR()
		return
	}
}

func (b *board) watchStatus(ctx context.Context) {
	if b.hook == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		last := readDeviceStatus()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.mu.Lock()
				if !b.sleeping && !b.closing {
					current := readDeviceStatus()
					oldURL := b.url
					qrErr := b.ensureConnectionQR()
					if qrErr != nil {
						b.reportError("refresh connection QR", qrErr)
					}
					if qrErr == nil && (current != last || oldURL != b.url) {
						var err error
						if b.showingQR {
							err = b.makeAndShowQR()
						} else {
							err = b.renderContent()
						}
						if err != nil {
							b.reportError("refresh status", err)
						}
						last = current
					}
				}
				b.mu.Unlock()
			}
		}
	}()
}
