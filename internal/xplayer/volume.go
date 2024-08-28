package xplayer

func NewVolume() Volume {
	return Volume{
		V: 100,
	}
}

type Volume struct {
	V int
}

func (v *Volume) Add(delta int) {
	volume := v.V + delta
	if volume < 0 {
		v.V = 0
	} else if volume > 100 {
		v.V = 100
	} else {
		v.V = volume
	}
}
