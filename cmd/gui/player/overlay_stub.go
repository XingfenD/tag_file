//go:build !(darwin && cgo) && !windows && !(linux && cgo)

package player

import (
	"fmt"
)

type stubOverlay struct{}

func newOverlay() (VideoOverlay, error) {
	return nil, fmt.Errorf("本平台不支持原生视频覆盖层")
}

func (o *stubOverlay) Create(x, y, w, h int) (uintptr, error) {
	return 0, fmt.Errorf("不支持")
}

func (o *stubOverlay) UpdatePosition(x, y, w, h int) {}

func (o *stubOverlay) Destroy() {}
