//go:build linux && cgo

package player

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <stdlib.h>

static Window createX11Window(Display *dpy, Window root, int x, int y, int w, int h) {
	Window win = XCreateSimpleWindow(dpy, root, x, y, w, h, 0, 0, 0);
	XMapWindow(dpy, win);
	XFlush(dpy);
	return win;
}
*/
import "C"
import (
	"fmt"
)

type linuxOverlay struct {
	display *C.Display
	window  C.Window
	root    C.Window
}

func newOverlay() (VideoOverlay, error) {
	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return nil, fmt.Errorf("无法打开 X display")
	}
	root := C.XDefaultRootWindow(dpy)
	return &linuxOverlay{display: dpy, root: root}, nil
}

func (o *linuxOverlay) Create(x, y, w, h int) (uintptr, error) {
	win := C.createX11Window(o.display, o.root, C.int(x), C.int(y), C.int(w), C.int(h))
	o.window = win
	return uintptr(win), nil
}

func (o *linuxOverlay) UpdatePosition(x, y, w, h int) {
	if o.window != 0 {
		C.XMoveResizeWindow(o.display, o.window, C.int(x), C.int(y), C.uint(w), C.uint(h))
	}
}

func (o *linuxOverlay) Destroy() {
	if o.window != 0 {
		C.XDestroyWindow(o.display, o.window)
		o.window = 0
	}
}
