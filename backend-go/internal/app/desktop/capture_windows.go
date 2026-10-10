//go:build windows

package desktop

import (
	"errors"
	"image"
	"image/png"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type nativeRect struct{ Left, Top, Right, Bottom int32 }
type bitmapInfo struct {
	Size                   uint32
	Width, Height          int32
	Planes, BitCount       uint16
	Compression, SizeImage uint32
	XPels, YPels           int32
	ClrUsed, ClrImportant  uint32
	Color                  uint32
}

// captureWindow records only the native application's visible window rectangle.
func captureWindow(hwnd uintptr, path string) error {
	var rect nativeRect
	ok, _, _ := user32.NewProc("GetWindowRect").Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 {
		return errors.New("cannot measure native window")
	}
	width, height := int(rect.Right-rect.Left), int(rect.Bottom-rect.Top)
	if width <= 0 || height <= 0 || width > 8192 || height > 8192 {
		return errors.New("invalid native window capture size")
	}
	screen, _, _ := user32.NewProc("GetDC").Call(0)
	defer user32.NewProc("ReleaseDC").Call(0, screen)
	gdi := windows.NewLazySystemDLL("gdi32.dll")
	dc, _, _ := gdi.NewProc("CreateCompatibleDC").Call(screen)
	defer gdi.NewProc("DeleteDC").Call(dc)
	bitmap, _, _ := gdi.NewProc("CreateCompatibleBitmap").Call(screen, uintptr(width), uintptr(height))
	defer gdi.NewProc("DeleteObject").Call(bitmap)
	if dc == 0 || bitmap == 0 {
		return errors.New("cannot allocate native capture bitmap")
	}
	old, _, _ := gdi.NewProc("SelectObject").Call(dc, bitmap)
	ok, _, _ = user32.NewProc("PrintWindow").Call(hwnd, dc, 2)
	gdi.NewProc("SelectObject").Call(dc, old)
	if ok == 0 {
		return errors.New("cannot copy native window pixels")
	}
	pixels := make([]byte, width*height*4)
	info := bitmapInfo{Size: 40, Width: int32(width), Height: -int32(height), Planes: 1, BitCount: 32}
	rows, _, _ := gdi.NewProc("GetDIBits").Call(dc, bitmap, 0, uintptr(height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&info)), 0)
	if rows != uintptr(height) {
		return errors.New("cannot read native window pixels")
	}
	for i := 0; i < len(pixels); i += 4 {
		pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
		pixels[i+3] = 255
	}
	output, err := os.Create(path)
	if err != nil {
		return err
	}
	defer output.Close()
	return png.Encode(output, &image.RGBA{Pix: pixels, Stride: width * 4, Rect: image.Rect(0, 0, width, height)})
}

func hasWindowIcon(hwnd uintptr) bool {
	icon, _, _ := user32.NewProc("SendMessageW").Call(hwnd, 0x7f, 1, 0)
	if icon == 0 {
		icon, _, _ = user32.NewProc("GetClassLongPtrW").Call(hwnd, ^uintptr(13))
	}
	return icon != 0
}
