package main

import (
	"fmt"
	"image"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GDI and user32 constants used by captureClient.
const (
	srcCopy      = 0x00cc0020
	captureBlt   = 0x40000000 // include layered (DWM-composed) windows in the copy
	dibRGBColors = 0
	biRGB        = 0
)

// The shot path uses only read-only queries (EnumWindows, GetWindowThreadProcessId,
// IsWindowVisible, IsIconic, GetClientRect, ClientToScreen) and a copy from the
// screen DC. It never sends a message to the game window (so no PrintWindow),
// never opens the process, and never sends input.
var (
	moduser32 = windows.NewLazySystemDLL("user32.dll")
	modgdi32  = windows.NewLazySystemDLL("gdi32.dll")

	procIsIconic               = moduser32.NewProc("IsIconic")
	procGetClientRect          = moduser32.NewProc("GetClientRect")
	procClientToScreen         = moduser32.NewProc("ClientToScreen")
	procGetDC                  = moduser32.NewProc("GetDC")
	procReleaseDC              = moduser32.NewProc("ReleaseDC")
	procCreateCompatibleDC     = modgdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modgdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = modgdi32.NewProc("SelectObject")
	procBitBlt                 = modgdi32.NewProc("BitBlt")
	procGetDIBits              = modgdi32.NewProc("GetDIBits")
	procDeleteObject           = modgdi32.NewProc("DeleteObject")
	procDeleteDC               = modgdi32.NewProc("DeleteDC")
)

type point struct{ X, Y int32 }

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// winCall invokes a user32 or gdi32 proc whose zero return means failure, and
// folds GetLastError into the error.
func winCall(proc *windows.LazyProc, args ...uintptr) (uintptr, error) {
	r, _, err := proc.Call(args...)
	if r == 0 {
		if err != nil && err != windows.ERROR_SUCCESS {
			return 0, fmt.Errorf("%s: %w", proc.Name, err)
		}
		return 0, fmt.Errorf("%s failed", proc.Name)
	}
	return r, nil
}

// enumPID and enumFound are the state of the current findGameWindow call.
// EnumWindows runs enumWindowsCB synchronously on the calling thread and only
// the sampling loop calls it, so package-level state is safe here.
var (
	enumPID   uint32
	enumFound []windows.HWND
)

var enumWindowsCB = windows.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
	var pid uint32
	windows.GetWindowThreadProcessId(hwnd, &pid)
	if pid == enumPID && windows.IsWindowVisible(hwnd) {
		enumFound = append(enumFound, hwnd)
	}
	return 1 // keep enumerating
})

// findGameWindow returns the visible top-level window of pid whose client area
// is 640x480, the size of th10 in windowed mode. Several candidates are normal
// (tool or helper windows), so the size decides which one is the game.
func findGameWindow(pid uint32) (windows.HWND, error) {
	enumPID, enumFound = pid, enumFound[:0]
	if err := windows.EnumWindows(enumWindowsCB, nil); err != nil {
		return 0, fmt.Errorf("EnumWindows: %w", err)
	}
	if len(enumFound) == 0 {
		return 0, fmt.Errorf("no visible top-level window for pid %d (the game must be open and not minimised)", pid)
	}
	var sizes []string
	for _, h := range enumFound {
		var rc windows.Rect
		if _, err := winCall(procGetClientRect, uintptr(h), uintptr(unsafe.Pointer(&rc))); err != nil {
			continue
		}
		w, hgt := rc.Right-rc.Left, rc.Bottom-rc.Top
		if w == 640 && hgt == 480 {
			return h, nil
		}
		sizes = append(sizes, fmt.Sprintf("%dx%d", w, hgt))
	}
	return 0, fmt.Errorf("no 640x480 window for pid %d among %d visible windows (client sizes: %s); use windowed mode at 100%% scaling",
		pid, len(enumFound), strings.Join(sizes, ", "))
}

// captureClient copies the client area of pid's game window from the screen as
// an RGBA image. frame is read just before and just after the copy; the returned
// values are those readings, so the file records the frames the screen spanned.
func captureClient(pid uint32, frame func() (int32, error)) (img *image.RGBA, before, after int32, err error) {
	hwnd, err := findGameWindow(pid)
	if err != nil {
		return nil, 0, 0, err
	}
	if minimised, _, _ := procIsIconic.Call(uintptr(hwnd)); minimised != 0 {
		return nil, 0, 0, fmt.Errorf("game window is minimised")
	}
	var rc windows.Rect
	if _, err := winCall(procGetClientRect, uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); err != nil {
		return nil, 0, 0, err
	}
	w, h := int(rc.Right-rc.Left), int(rc.Bottom-rc.Top)
	var origin point // client (0,0) in screen coordinates
	if _, err := winCall(procClientToScreen, uintptr(hwnd), uintptr(unsafe.Pointer(&origin))); err != nil {
		return nil, 0, 0, err
	}

	screen, err := winCall(procGetDC, 0)
	if err != nil {
		return nil, 0, 0, err
	}
	defer procReleaseDC.Call(0, screen)
	mem, err := winCall(procCreateCompatibleDC, screen)
	if err != nil {
		return nil, 0, 0, err
	}
	defer procDeleteDC.Call(mem)
	bmp, err := winCall(procCreateCompatibleBitmap, screen, uintptr(w), uintptr(h))
	if err != nil {
		return nil, 0, 0, err
	}
	defer procDeleteObject.Call(bmp)
	old, err := winCall(procSelectObject, mem, bmp)
	if err != nil {
		return nil, 0, 0, err
	}

	// No early return between selecting the bitmap and deselecting it below;
	// errors from the frame reads are kept and reported after the cleanup.
	before, errBefore := frame()
	_, errBlt := winCall(procBitBlt, mem, 0, 0, uintptr(w), uintptr(h),
		screen, uintptr(origin.X), uintptr(origin.Y), srcCopy|captureBlt)
	after, errAfter := frame()
	procSelectObject.Call(mem, old) // GetDIBits must not read a bitmap that is selected

	if errBefore != nil {
		return nil, 0, 0, errBefore
	}
	if errAfter != nil {
		return nil, 0, 0, errAfter
	}
	if errBlt != nil {
		return nil, 0, 0, errBlt
	}

	bi := bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       int32(w),
		Height:      -int32(h), // negative: top-down rows
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}
	pix := make([]byte, w*h*4) // BGRA
	if _, err := winCall(procGetDIBits, mem, bmp, 0, uintptr(h),
		uintptr(unsafe.Pointer(&pix[0])), uintptr(unsafe.Pointer(&bi)), dibRGBColors); err != nil {
		return nil, 0, 0, err
	}

	img = image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pix); i += 4 {
		img.Pix[i+0] = pix[i+2] // R
		img.Pix[i+1] = pix[i+1] // G
		img.Pix[i+2] = pix[i+0] // B
		img.Pix[i+3] = 0xff     // the screen copy has no alpha
	}
	return img, before, after, nil
}
