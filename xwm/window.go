package xwm

import "github.com/jezek/xgb/xproto"

type Window interface {
	Mute(mute bool) error
	Start(url string) error
	Stop() error
	KeyPress(ev xproto.KeyPressEvent)
	ButtonPress(ev xproto.ButtonPressEvent)
	Release()
}

type WindowFactory func(wid xproto.Window) (Window, error)
