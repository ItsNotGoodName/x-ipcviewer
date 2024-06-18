package xwmold

import (
	"fmt"
	"slices"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/thejerf/suture/v4"
)

type State struct {
	WID            xproto.Window
	Width          uint16
	Height         uint16
	Panes          []StatePane
	FullscreenUUID string
	View           string
	ViewManual     []StateViewManual
}

type StatePane struct {
	UUID    string
	WID     xproto.Window
	Service suture.ServiceToken
}

type StateViewManual struct {
	X int16
	Y int16
	W uint16
	H uint16
}

func (s State) Render(conn *xgb.Conn) error {
	if err := s.RenderFullscreen(conn); err != nil {
		return err
	}
	if err := s.RenderLayout(conn); err != nil {
		return err
	}
	return nil
}

func (s State) RenderFullscreen(conn *xgb.Conn) error {
	idx := slices.IndexFunc(s.Panes, func(p StatePane) bool { return p.UUID == s.FullscreenUUID })
	if idx == -1 {
		return nil
	}

	x, y, w, h := int16(0), int16(0), s.Width, s.Height

	err := xproto.ConfigureWindowChecked(conn, s.Panes[idx].WID,
		xproto.ConfigWindowStackMode|xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{0, uint32(x), uint32(y), uint32(w), uint32(h)}).
		Check()
	if err != nil {
		return err
	}

	return nil
}

func (s State) RenderLayout(conn *xgb.Conn) error {
	if s.View != "auto" {
		return fmt.Errorf("view %s not supported", s.View)
	}

	layout := NewLayoutGrid(s.Width, s.Height, len(s.Panes))
	for i := range s.Panes {
		x, y, w, h := layout.Pane(i)

		err := xproto.ConfigureWindowChecked(conn, s.Panes[i].WID,
			xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
			[]uint32{uint32(x), uint32(y), uint32(w), uint32(h)}).
			Check()
		if err != nil {
			return err
		}
	}

	return nil
}
