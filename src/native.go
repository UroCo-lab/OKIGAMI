package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func (b *board) reportError(where string, err error) { log.Printf("%s: %v", where, err) }

func (b *board) buttonHit(x, y, screenW, screenH int) bool {
	return b.controlHit(x, y, screenW, screenH) == 1
}

// 0 = content, 1 = QR, 2 = close. Both controls rotate with the content.
func (b *board) controlHit(x, y, screenW, screenH int) int {
	// PW4 X11 is portrait. Keep a mapping for a landscape X11 root as well.
	px, py := x*b.width/screenW, y*b.height/screenH
	if screenW > screenH {
		px = y * b.width / screenH
		py = (screenW - 1 - x) * b.height / screenW
	}
	w, h := b.width, b.height
	if b.landscape {
		px, py = py, b.width-1-px
		w, h = b.height, b.width
	}
	point := image.Pt(px, py)
	if point.In(controlRect(w, h).Inset(-12)) {
		return 1
	}
	if point.In(exitRect(w, h).Inset(-12)) {
		return 2
	}
	return 0
}

func (b *board) startNative(ctx context.Context) (func(), error) {
	if b.hook == "" {
		return func() {}, nil
	}
	conn, err := xgb.NewConnDisplay(":0")
	if err != nil {
		return nil, fmt.Errorf("X11: %w", err)
	}
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	if int(screen.WidthInPixels) != b.width || int(screen.HeightInPixels) != b.height {
		conn.Close()
		return nil, fmt.Errorf("unexpected X11 size %dx%d", screen.WidthInPixels, screen.HeightInPixels)
	}
	var format xproto.Format
	for _, f := range setup.PixmapFormats {
		if f.Depth == screen.RootDepth {
			format = f
			break
		}
	}
	var visual xproto.VisualInfo
	for _, depth := range screen.AllowedDepths {
		for _, v := range depth.Visuals {
			if v.VisualId == screen.RootVisual {
				visual = v
			}
		}
	}
	pixmap, err := xproto.NewPixmapId(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err = xproto.CreatePixmapChecked(conn, screen.RootDepth, pixmap, xproto.Drawable(screen.Root), screen.WidthInPixels, screen.HeightInPixels).Check(); err != nil {
		conn.Close()
		return nil, err
	}
	gc, err := xproto.NewGcontextId(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err = xproto.CreateGCChecked(conn, gc, xproto.Drawable(pixmap), xproto.GcForeground, []uint32{screen.WhitePixel}).Check(); err != nil {
		conn.Close()
		return nil, err
	}
	xproto.PolyFillRectangle(conn, xproto.Drawable(pixmap), gc, []xproto.Rectangle{{Width: screen.WidthInPixels, Height: screen.HeightInPixels}})
	window, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	mask := uint32(xproto.CwBackPixmap | xproto.CwOverrideRedirect | xproto.CwEventMask)
	values := []uint32{uint32(pixmap), 1, xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskExposure}
	err = xproto.CreateWindowChecked(conn, screen.RootDepth, window, screen.Root, 0, 0, screen.WidthInPixels, screen.HeightInPixels, 0, xproto.WindowClassInputOutput, screen.RootVisual, mask, values).Check()
	if err != nil {
		conn.Close()
		return nil, err
	}
	var paintMu sync.Mutex
	mapped := false
	b.nativePaint = func(path string) error {
		paintMu.Lock()
		defer paintMu.Unlock()
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			return err
		}
		if img.Bounds().Dx() != b.width || img.Bounds().Dy() != b.height {
			return fmt.Errorf("image does not match X11 panel")
		}
		pixels, stride, err := packXPixels(img, format.BitsPerPixel, format.ScanlinePad, setup.ImageByteOrder, screen.WhitePixel, screen.BlackPixel, visual.RedMask, visual.GreenMask, visual.BlueMask)
		if err != nil {
			return err
		}
		rows := (int(setup.MaximumRequestLength)*4 - 64) / stride
		if rows > 32 {
			rows = 32
		}
		if rows < 1 {
			return fmt.Errorf("X11 request limit too small")
		}
		for y := 0; y < b.height; y += rows {
			count := min(rows, b.height-y)
			err = xproto.PutImageChecked(conn, xproto.ImageFormatZPixmap, xproto.Drawable(pixmap), gc, screen.WidthInPixels, uint16(count), 0, int16(y), 0, screen.RootDepth, pixels[y*stride:(y+count)*stride]).Check()
			if err != nil {
				return err
			}
		}
		// Keep X11's own backing pixels identical to the displayed e-ink image.
		if err = xproto.ClearAreaChecked(conn, false, window, 0, 0, screen.WidthInPixels, screen.HeightInPixels).Check(); err != nil {
			return err
		}
		conn.Sync()
		// Map only after the first image is ready; never expose a white window.
		if !mapped {
			if err = xproto.MapWindowChecked(conn, window).Check(); err != nil {
				return err
			}
			conn.Sync()
			mapped = true
		}
		return nil
	}
	b.nativeShow = func(show bool) {
		if show {
			xproto.MapWindow(conn, window)
		} else {
			xproto.UnmapWindow(conn, window)
		}
		conn.Sync()
	}
	name := "OKIGAMI"
	xproto.ChangeProperty(conn, xproto.PropModeReplace, window, xproto.AtomWmName, xproto.AtomString, 8, uint32(len(name)), []byte(name))
	log.Printf("Native controls ready: %dx%d depth=%d bpp=%d pad=%d white=%d black=%d", screen.WidthInPixels, screen.HeightInPixels, screen.RootDepth, format.BitsPerPixel, format.ScanlinePad, screen.WhitePixel, screen.BlackPixel)
	go func() {
		var pressedControl int
		var lastDraw time.Time
		for {
			event, xerr := conn.WaitForEvent()
			if event == nil && xerr == nil {
				return
			}
			if xerr != nil {
				log.Printf("X11 event: %v", xerr)
				continue
			}
			switch e := event.(type) {
			case xproto.ButtonPressEvent:
				b.mu.Lock()
				pressedControl = 0
				if !b.sleeping && !b.loading {
					pressedControl = b.controlHit(int(e.EventX), int(e.EventY), int(screen.WidthInPixels), int(screen.HeightInPixels))
				}
				b.mu.Unlock()
			case xproto.ButtonReleaseEvent:
				b.mu.Lock()
				releasedControl := 0
				awake := !b.sleeping && !b.loading
				if awake {
					releasedControl = b.controlHit(int(e.EventX), int(e.EventY), int(screen.WidthInPixels), int(screen.HeightInPixels))
					b.noteActivityLocked(time.Now())
				}
				b.mu.Unlock()
				if awake && pressedControl != 0 && pressedControl == releasedControl {
					b.toolbarTap(releasedControl)
				}
				pressedControl = 0
			case xproto.ExposeEvent:
				if e.Count == 0 && time.Since(lastDraw) > 500*time.Millisecond {
					lastDraw = time.Now()
					go func() {
						time.Sleep(200 * time.Millisecond)
						b.mu.Lock()
						defer b.mu.Unlock()
						if b.closing || b.loading {
							return
						}
						if b.sleeping {
							if err := b.action("display", b.dir+"/sleep/board.png"); err != nil {
								b.reportError("sleep expose", err)
							}
							return
						}
						var err error
						if b.showingQR {
							err = b.action("qr", b.dir+"/qr.png")
						} else {
							err = b.renderContent()
						}
						if err != nil {
							b.reportError("expose redraw", err)
						}
					}()
				}
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	return func() {
		xproto.DestroyWindow(conn, window)
		xproto.FreeGC(conn, gc)
		xproto.FreePixmap(conn, pixmap)
		conn.Sync()
		conn.Close()
	}, nil
}
