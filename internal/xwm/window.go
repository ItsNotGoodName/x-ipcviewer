package xwm

import (
	"github.com/ItsNotGoodName/x-ipcviewer/internal/xcursor"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Window struct {
	WID    xproto.Window
	Width  uint16
	Height uint16
}

func CreateWindow(conn *xgb.Conn) (Window, error) {
	screen := xproto.Setup(conn).DefaultScreen(conn)

	cursor, err := xcursor.CreateCursor(conn, xcursor.LeftPtr)
	if err != nil {
		return Window{}, err
	}

	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		return Window{}, err
	}

	if err := xproto.CreateWindowChecked(conn, screen.RootDepth,
		wid, screen.Root,
		0, 0, screen.WidthInPixels, screen.HeightInPixels, 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask|xproto.CwCursor, // 1, 2, 3
		[]uint32{
			0, // 1
			xproto.EventMaskStructureNotify | xproto.EventMaskKeyPress | xproto.EventMaskButtonPress, // 2
			uint32(cursor), // 3
		}).Check(); err != nil {
		return Window{}, err
	}

	if err := xproto.MapWindowChecked(conn, wid).Check(); err != nil {
		return Window{}, err
	}

	return Window{
		WID:    wid,
		Width:  screen.WidthInPixels,
		Height: screen.HeightInPixels,
	}, nil
}

func CreateSubWindow(conn *xgb.Conn, root xproto.Window, x, y int16, w, h uint16) (xproto.Window, error) {
	// Generate window id
	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		return 0, err
	}

	// Create window in root
	if err := xproto.CreateWindowChecked(conn, xproto.WindowClassCopyFromParent,
		wid, root,
		x, y, w, h, 0,
		xproto.WindowClassInputOutput, xproto.WindowClassCopyFromParent, 0, []uint32{}).Check(); err != nil {
		return 0, err
	}

	return wid, err
}

func DestroySubWindow(conn *xgb.Conn, wid xproto.Window) {
	xproto.DestroySubwindows(conn, wid).Check()
}
