package mosaic

type LayoutManualWindow struct {
	X float32
	Y float32
	W float32
	H float32
}

type LayoutManual struct {
	windows []LayoutManualWindow
}

func NewLayoutManual(windows []LayoutManualWindow) LayoutManual {
	return LayoutManual{
		windows: windows,
	}
}

func (l LayoutManual) Count() int {
	return len(l.windows)
}

func (l LayoutManual) update(wins []Window, w, h uint16) {
	for i := range wins {
		wins[i].X = uint16(l.windows[i].X * float32(w))
		wins[i].Y = uint16(l.windows[i].Y * float32(h))
		wins[i].W = uint16((l.windows[i].W+l.windows[i].X)*float32(w)) - wins[i].X
		wins[i].H = uint16((l.windows[i].H+l.windows[i].Y)*float32(h)) - wins[i].Y
	}
}
