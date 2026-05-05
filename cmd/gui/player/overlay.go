package player

type VideoOverlay interface {
	Create(x, y, w, h int) (uintptr, error)
	UpdatePosition(x, y, w, h int)
	Destroy()
}
