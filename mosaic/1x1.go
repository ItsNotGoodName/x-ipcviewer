package mosaic

type Layout1x1 struct{}

func (l Layout1x1) Count() int {
	return 1
}

func (l Layout1x1) Update(wins []Window, w, h uint16) {
	wins[0].X, wins[0].Y, wins[0].Width, wins[0].Height = 0, 0, w, h
}
