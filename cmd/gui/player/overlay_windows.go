//go:build windows

package player

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW     = user32.NewProc("FindWindowW")
	procCreateWindowExW = user32.NewProc("CreateWindowExW")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
	procSetWindowPos    = user32.NewProc("SetWindowPos")
)

const (
	WS_CHILD   = 0x40000000
	WS_VISIBLE = 0x10000000
	SWP_NOZORDER = 0x0004
)

type windowsOverlay struct {
	hwnd   windows.Handle
	parent windows.Handle
}

func newOverlay() (VideoOverlay, error) {
	title, _ := syscall.UTF16PtrFromString("TagFile - File Tag Manager")
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return nil, fmt.Errorf("无法找到主窗口")
	}
	return &windowsOverlay{parent: windows.Handle(hwnd)}, nil
}

func (o *windowsOverlay) Create(x, y, w, h int) (uintptr, error) {
	className, _ := syscall.UTF16PtrFromString("static")
	windowName, _ := syscall.UTF16PtrFromString("")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		WS_CHILD|WS_VISIBLE,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(o.parent),
		0, 0, 0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowEx 失败")
	}
	o.hwnd = windows.Handle(hwnd)
	return hwnd, nil
}

func (o *windowsOverlay) UpdatePosition(x, y, w, h int) {
	if o.hwnd != 0 {
		procSetWindowPos.Call(
			uintptr(o.hwnd), 0,
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			SWP_NOZORDER,
		)
	}
}

func (o *windowsOverlay) Destroy() {
	if o.hwnd != 0 {
		procDestroyWindow.Call(uintptr(o.hwnd))
		o.hwnd = 0
	}
}
