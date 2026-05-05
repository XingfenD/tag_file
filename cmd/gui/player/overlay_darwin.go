//go:build darwin && cgo

package player

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void* getMainContentView() {
	NSArray *windows = [NSApp windows];
	for (NSWindow *window in windows) {
		if (![window isVisible]) continue;
		if ([window isMiniaturized]) continue;
		// Wails uses a regular NSWindow, not NSPanel
		if ([window isKindOfClass:[NSPanel class]]) continue;
		return (void*)CFBridgingRetain([window contentView]);
	}
	return NULL;
}

static void* createOverlayView(void* parentPtr, int x, int y, int w, int h) {
	NSView *parent = (__bridge NSView*)parentPtr;
	NSView *view = [[NSView alloc] initWithFrame:NSMakeRect(x, y, w, h)];
	[view setWantsLayer:YES];
	view.layer.backgroundColor = [[NSColor blackColor] CGColor];
	[parent addSubview:view positioned:NSWindowAbove relativeTo:nil];
	return (void*)CFBridgingRetain(view);
}

static void updateOverlayFrame(void* viewPtr, int x, int y, int w, int h) {
	NSView *view = (__bridge NSView*)viewPtr;
	view.frame = NSMakeRect(x, y, w, h);
}

static void removeOverlayView(void* viewPtr) {
	if (viewPtr == NULL) return;
	NSView *view = (__bridge NSView*)viewPtr;
	[view removeFromSuperview];
	CFRelease(viewPtr);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type darwinOverlay struct {
	view   unsafe.Pointer // NSView*
	parent unsafe.Pointer // contentView NSView*
}

func newOverlay() (VideoOverlay, error) {
	parent := C.getMainContentView()
	if parent == nil {
		return nil, fmt.Errorf("无法获取主窗口 contentView")
	}
	return &darwinOverlay{parent: parent}, nil
}

func (o *darwinOverlay) Create(x, y, w, h int) (uintptr, error) {
	view := C.createOverlayView(o.parent, C.int(x), C.int(y), C.int(w), C.int(h))
	if view == nil {
		return 0, fmt.Errorf("无法创建覆盖 NSView")
	}
	o.view = view
	return uintptr(view), nil
}

func (o *darwinOverlay) UpdatePosition(x, y, w, h int) {
	if o.view != nil {
		C.updateOverlayFrame(o.view, C.int(x), C.int(y), C.int(w), C.int(h))
	}
}

func (o *darwinOverlay) Destroy() {
	if o.view != nil {
		C.removeOverlayView(o.view)
		o.view = nil
	}
}
