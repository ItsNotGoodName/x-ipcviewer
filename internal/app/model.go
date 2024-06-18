package app

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/xwm"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func (m Model) Init(conn *xgb.Conn) (xwm.Model, xwm.Cmd) {
	window, err := xwm.CreateWindow(conn)
	if err != nil {
		return m, xwm.Quit
	}

	m.FullscreenUUID = "f"
	m.View = "auto"
	m.WID = window.WID
	m.Width = window.Width
	m.Height = window.Height

	return m, nil
}

func (m Model) Update(conn *xgb.Conn, msg xwm.Msg) (xwm.Model, xwm.Cmd) {
	switch ev := msg.(type) {
	case xproto.ConfigureNotifyEvent:
		slog.Debug("ConfigureNotifyEvent:", "event", ev)

		if ev.Window == m.WID {
			m.Width = ev.Width
			m.Height = ev.Height
		}

		return m, nil
	case xproto.ButtonPressEvent:
		slog.Debug("ButtonPressEvent", "detail", ev.Detail)

		return m, nil
	case xproto.KeyPressEvent:
		slog.Debug("KeyPressEvent", "detail", ev.Detail)

		if ev.Detail == 24 {
			slog.Debug("exit: quit key pressed")
			return m, xwm.Quit
		}

		if ev.Detail == 65 {
			sub, err := xwm.CreateSubWindow(conn, m.WID, 0, 0, 1, 1)
			if err != nil {
				fmt.Println(err)
				return m, nil
			}

			window, err := NewWindow(sub.WID)
			if err != nil {
				return m, nil
			}

			go window.Serve(context.Background())

			m.Panes = append(m.Panes, ModelPane{
				UUID:   "",
				Window: window,
			})

			return m, nil
		}

		return m, nil
	case xproto.DestroyNotifyEvent:
		// Depending on the user's desktop environment (especially
		// window manager), killing a window might close the
		// client's X connection (e. g. the default Ubuntu
		// desktop environment).
		//
		// If that's the case for your environment, closing this example's window
		// will also close the underlying Go program (because closing the X
		// connection gives a nil event and EOF error).
		//
		// Consider how a single application might have multiple windows
		// (e.g. an open popup or dialog, ...)
		//
		// With other DEs, the X connection will still stay open even after the
		// X window is closed. For these DEs (e.g. i3) we have to check whether
		// the WM sent us a DestroyNotifyEvent and close our program.
		//
		// For more information about closing windows while maintaining
		// the X connection see
		// https://github.com/jezek/xgbutil/blob/master/_examples/graceful-window-close/main.go
		slog.Debug("exit: destroy notify event")

		return m, xwm.Quit
	default:
		slog.Debug("unknown event", "event", ev)
		return m, nil
	}
}

type Model struct {
	WID            xproto.Window
	Width          uint16
	Height         uint16
	Panes          []ModelPane
	FullscreenUUID string
	View           string
	ViewManual     []ModelViewManual
}

type ModelPane struct {
	UUID   string
	Window Window
}

type ModelViewManual struct {
	X int16
	Y int16
	W uint16
	H uint16
}

func (m Model) Render(conn *xgb.Conn) error {
	idx := slices.IndexFunc(m.Panes, func(p ModelPane) bool { return p.UUID == m.FullscreenUUID })

	if idx == -1 {
		if m.View != "auto" {
			return fmt.Errorf("view %s not supported", m.View)
		}

		layout := NewLayoutGrid(m.Width, m.Height, len(m.Panes))
		for i := range m.Panes {
			x, y, w, h := layout.Pane(i)

			err := xproto.ConfigureWindowChecked(conn, m.Panes[i].Window.wid,
				xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
				[]uint32{uint32(x), uint32(y), uint32(w), uint32(h)}).
				Check()
			if err != nil {
				return err
			}
		}

		return nil
	} else {
		x, y, w, h := int16(0), int16(0), m.Width, m.Height
		fmt.Println(x, y, w, h)

		err := xproto.ConfigureWindowChecked(conn, m.Panes[idx].Window.wid,
			xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
			[]uint32{uint32(x), uint32(y), uint32(w), uint32(h)}).
			Check()
		if err != nil {
			return err
		}

		return nil
	}
}
