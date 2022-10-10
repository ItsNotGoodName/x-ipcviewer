package mosaic

type LayoutGrid struct {
	xc int
	yc int
}

func NewLayoutGridCount(count int) LayoutGrid {
	xc, yc := 0, 0
	for xc*yc < count {
		xc++
		if xc*yc >= count {
			break
		}
		yc++
	}

	return NewLayoutGrid(xc, yc)
}

func NewLayoutGrid(xc, yc int) LayoutGrid {
	return LayoutGrid{
		xc: xc,
		yc: yc,
	}
}

func (l LayoutGrid) Count() int {
	return l.xc * l.yc
}

func (l LayoutGrid) update(wins []Window, w, h uint16) {
	fw := uint16(float32(w) * (1.0 / float32(l.xc)))
	fh := uint16(float32(h) * (1.0 / float32(l.yc)))

	for i := 0; i < l.yc; i++ {
		fy := fh * uint16(i)
		for j := 0; j < l.xc; j++ {
			fx := fw * uint16(j)
			idx := (i * l.xc) + j
			wins[idx].X, wins[idx].Y, wins[idx].W, wins[idx].H = fx, fy, fw, fh
		}
	}
}
