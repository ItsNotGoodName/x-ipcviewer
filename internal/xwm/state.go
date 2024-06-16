package xwm

import (
	"github.com/jezek/xgb/xproto"
	"github.com/thejerf/suture/v4"
)

type State struct {
	WID            xproto.Window
	Width          uint16
	Height         uint16
	Panes          []StatePane
	FullscreenUUID string
}

type StatePane struct {
	UUID    string
	WID     xproto.Window
	Service suture.ServiceToken
	X       int16
	Y       int16
	W       uint16
	H       uint16
}

func (s *State) UpdateWindow(width, height uint16) {
	s.Width = width
	s.Height = height
}
