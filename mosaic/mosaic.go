package mosaic

type (
	Window struct {
		X      uint16
		Y      uint16
		Width  uint16
		Height uint16
	}

	Mosaic struct {
		windows []Window
		layout  Layout
	}

	Layout interface {
		Count() int
		Update(wins []Window, w, h uint16)
	}
)

func NewMosaic(layout Layout) Mosaic {
	m := Mosaic{}
	m.SetLayout(layout)
	return m
}

func (m *Mosaic) SetLayout(layout Layout) {
	m.layout = layout
	m.windows = make([]Window, layout.Count())
}

func (m *Mosaic) Windows(w, h uint16) []Window {
	m.layout.Update(m.windows, w, h)
	return m.windows
}
