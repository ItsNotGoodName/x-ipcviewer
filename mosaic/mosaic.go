package mosaic

type (
	Window struct {
		X uint16
		Y uint16
		W uint16
		H uint16
	}

	Mosaic struct {
		windows []Window
		layout  Layout
	}

	Layout interface {
		Count() int
		update(wins []Window, w, h uint16)
	}
)

func New(layout Layout) Mosaic {
	return Mosaic{
		layout:  layout,
		windows: make([]Window, layout.Count()),
	}
}

func (m Mosaic) Windows(w, h uint16) []Window {
	m.layout.update(m.windows, w, h)
	return m.windows
}
