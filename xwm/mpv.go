package xwm

import (
	"strconv"

	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb/xproto"
)

func NewMPV(wid xproto.Window) *mpv.Mpv {
	m := mpv.New()
	m.SetOptionString("wid", strconv.Itoa(int(wid)))
	return m
}
