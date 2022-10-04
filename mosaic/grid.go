package mosaic

type LayoutGrid struct {
	xc int
	yc int
	c  int
}

func NewLayoutGrid(wc, hc int) LayoutGrid {
	return LayoutGrid{
		xc: wc,
		yc: hc,
		c:  wc * hc,
	}
}

func (l LayoutGrid) Count() int {
	return l.c
}

func (l LayoutGrid) Update(wins []Window, w, h uint16) {
	fw := uint16(float32(w) * (1.0 / float32(l.xc)))
	fh := uint16(float32(h) * (1.0 / float32(l.yc)))

	for i := 0; i < l.yc; i++ {
		fy := fh * uint16(i)
		for j := 0; j < l.xc; j++ {
			fx := fw * uint16(j)
			idx := (i * l.xc) + j
			wins[idx].X, wins[idx].Y, wins[idx].Width, wins[idx].Height = fx, fy, fw, fh
		}
	}
}
