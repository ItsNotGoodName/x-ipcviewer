package mosaic

type Layout2x2 struct{}

func (l Layout2x2) Count() int {
	return 4
}

func (l Layout2x2) Update(wins []Window, w, h uint16) {
	hw, hh := w/2, h/2
	wins[0].X, wins[0].Y, wins[0].Width, wins[0].Height = 0, 0, hw, hh
	wins[1].X, wins[1].Y, wins[1].Width, wins[1].Height = hw, 0, hw, hh
	wins[2].X, wins[2].Y, wins[2].Width, wins[2].Height = 0, hh, hw, hh
	wins[3].X, wins[3].Y, wins[3].Width, wins[3].Height = hw, hh, hw, hh
}
