package executor

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

var (
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	procCreateCompatDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatBM = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject   = gdi32.NewProc("SelectObject")
	procBitBlt         = gdi32.NewProc("BitBlt")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
	procDeleteDC       = gdi32.NewProc("DeleteDC")
	procGetDIBits      = gdi32.NewProc("GetDIBits")

	procGetDC      = user32.NewProc("GetDC")
	procReleaseDC  = user32.NewProc("ReleaseDC")
	procGetCursorInfo = user32.NewProc("GetCursorInfo")
	procDrawIconEx    = user32.NewProc("DrawIconEx")
)

const (
	srccopy        = 0x00CC0020
	dibRgbColors   = 0
	biRgb          = 0
	cursorShowing  = 0x00000001
	diFormal       = 0x0003
)

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type bitmapInfo struct {
	bmiHeader bitmapInfoHeader
}

type cursorInfo struct {
	cbSize      uint32
	flags       uint32
	hCursor     uintptr
	ptScreenPos point
}

type point struct {
	x, y int32
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

var procGetIconInfo = user32.NewProc("GetIconInfo")

func captureScreenWithCursor() (image.Image, error) {
	sw, _, _ := procGetSystemMetrics.Call(smCxscreen)
	sh, _, _ := procGetSystemMetrics.Call(smCyscreen)
	w := int(sw)
	h := int(sh)

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, hdc)

	memDC, _, _ := procCreateCompatDC.Call(hdc)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(memDC)

	hBitmap, _, _ := procCreateCompatBM.Call(hdc, sw, sh)
	if hBitmap == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer procDeleteObject.Call(hBitmap)

	procSelectObject.Call(memDC, hBitmap)
	procBitBlt.Call(memDC, 0, 0, sw, sh, hdc, 0, 0, srccopy)

	// Draw cursor onto the captured bitmap
	var ci cursorInfo
	ci.cbSize = uint32(unsafe.Sizeof(ci))
	ret, _, _ := procGetCursorInfo.Call(uintptr(unsafe.Pointer(&ci)))
	if ret != 0 && ci.flags == cursorShowing && ci.hCursor != 0 {
		curX := int(ci.ptScreenPos.x)
		curY := int(ci.ptScreenPos.y)

		var ii iconInfo
		procGetIconInfo.Call(ci.hCursor, uintptr(unsafe.Pointer(&ii)))
		if ii.hbmMask != 0 {
			defer procDeleteObject.Call(ii.hbmMask)
		}
		if ii.hbmColor != 0 {
			defer procDeleteObject.Call(ii.hbmColor)
		}

		drawX := curX - int(ii.xHotspot)
		drawY := curY - int(ii.yHotspot)
		procDrawIconEx.Call(memDC,
			uintptr(drawX), uintptr(drawY),
			ci.hCursor,
			0, 0, 0, 0,
			diFormal,
		)
	}

	// Read pixels via GetDIBits (bottom-up BGR -> top-down RGBA)
	bi := bitmapInfo{
		bmiHeader: bitmapInfoHeader{
			biSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			biWidth:       int32(w),
			biHeight:      -int32(h), // top-down
			biPlanes:      1,
			biBitCount:    32,
			biCompression: biRgb,
		},
	}

	pixels := make([]byte, w*h*4)
	ret2, _, _ := procGetDIBits.Call(memDC, hBitmap, 0, uintptr(h),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bi)),
		dibRgbColors,
	)
	if ret2 == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}

	// Convert BGRA -> RGBA
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pixels); i += 4 {
		img.Pix[i+0] = pixels[i+2] // R
		img.Pix[i+1] = pixels[i+1] // G
		img.Pix[i+2] = pixels[i+0] // B
		img.Pix[i+3] = 255         // A
	}

	return img, nil
}
