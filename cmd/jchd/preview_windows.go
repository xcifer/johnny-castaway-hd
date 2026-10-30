package main

import (
	"image"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procBeginPaint       = user32.NewProc("BeginPaint")
	procEndPaint         = user32.NewProc("EndPaint")
	procFillRect         = user32.NewProc("FillRect")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procStretchDIBits    = gdi32.NewProc("StretchDIBits")
	procSetStretchMode   = gdi32.NewProc("SetStretchBltMode")
)

const (
	wmDestroy = 0x2
	wmPaint   = 0xF
)

type rect struct{ left, top, right, bottom int32 }

// showPreview draws the title picture into the little screen of Screen
// Saver Settings, the window parent, until Windows closes it.
func showPreview(parent uintptr) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const perMonitorAwareV2 = ^uintptr(3)
	procSetProcessDpiAwarenessContext.Call(perMonitorAwareV2)

	var pic *image.RGBA
	s := loadSettings()
	dir, err := s.DataDir, error(nil)
	if dir == "" {
		dir, err = defaultDataDir()
	}
	if err == nil {
		pic, _ = previewImage(dir) // without the game files the preview stays black
	}
	var bgra []byte
	if pic != nil {
		w, h := previewCrop.Dx(), previewCrop.Dy()
		bgra = make([]byte, 4*w*h)
		for y := range h {
			for x := range w {
				c := pic.RGBAAt(previewCrop.Min.X+x, previewCrop.Min.Y+y)
				copy(bgra[4*(y*w+x):], []byte{c.B, c.G, c.R, 0})
			}
		}
	}

	proc := func(h, msg, wp, lp uintptr) uintptr {
		switch msg {
		case wmPaint:
			var ps struct {
				hdc       uintptr
				erase     int32
				paint     rect
				restore   int32
				incUpdate int32
				reserved  [32]byte
			}
			dc, _, _ := procBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
			var r rect
			procGetClientRect.Call(h, uintptr(unsafe.Pointer(&r)))
			black, _, _ := procGetStockObject.Call(4) // BLACK_BRUSH
			procFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), black)
			if bgra != nil {
				drawPreview(dc, r, bgra)
			}
			procEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
			return 0
		case wmDestroy:
			procPostQuitMessage.Call(0)
			return 0
		}
		ret, _, _ := procDefWindowProcW.Call(h, msg, wp, lp)
		return ret
	}

	module, _, _ := procGetModuleHandleW.Call(0)
	name, _ := windows.UTF16PtrFromString("JohnnyCastawayHDPreview")
	class := struct {
		size                uint32
		style               uint32
		proc                uintptr
		clsExtra, wndExtra  int32
		instance, icon      uintptr
		cursor, background  uintptr
		menuName, className *uint16
		smallIcon           uintptr
	}{proc: windows.NewCallback(proc), instance: module, className: name}
	class.size = uint32(unsafe.Sizeof(class))
	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class))); r == 0 {
		return
	}
	var r rect
	procGetClientRect.Call(parent, uintptr(unsafe.Pointer(&r)))
	h, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(name)), 0, wsChild|wsVisible,
		0, 0, uintptr(r.right), uintptr(r.bottom), parent, 0, module, 0)
	if h == 0 {
		return // no such window to draw into
	}
	var msg [48]byte // MSG
	for {
		if r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0); int32(r) <= 0 {
			return
		}
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// drawPreview draws the cropped title picture, bgra, as large as fits in
// r, centred.
func drawPreview(dc uintptr, r rect, bgra []byte) {
	sw, sh := previewCrop.Dx(), previewCrop.Dy()
	w, h := int(r.right-r.left), int(r.bottom-r.top)
	dw, dh := w, w*sh/sw
	if dh > h {
		dw, dh = h*sw/sh, h
	}
	info := struct {
		size                 uint32
		width, height        int32
		planes, bitCount     uint16
		compression, imgSize uint32
		xPerM, yPerM         int32
		used, important      uint32
	}{width: int32(sw), height: -int32(sh), planes: 1, bitCount: 32} // negative height: top row first
	info.size = uint32(unsafe.Sizeof(info))
	procSetStretchMode.Call(dc, 4) // HALFTONE: averages pixels when shrinking
	procStretchDIBits.Call(dc, uintptr((w-dw)/2), uintptr((h-dh)/2), uintptr(dw), uintptr(dh),
		0, 0, uintptr(sw), uintptr(sh), uintptr(unsafe.Pointer(&bgra[0])), uintptr(unsafe.Pointer(&info)),
		0, 0x00CC0020) // DIB_RGB_COLORS, SRCCOPY
}
