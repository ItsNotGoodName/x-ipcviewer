package xwmold

import (
	"log/slog"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/xwm"
	"github.com/jezek/xgb/xproto"
)

type Main struct {
	State
}

func (m Main) Init() {
}

func (m Main) Update(msg xwm.Msg) (xwm.Model, xwm.Render) {
	switch ev := msg.(type) {
	case xproto.ConfigureNotifyEvent:
		slog.Debug("ConfigureNotifyEvent:", "event", ev)

		return m, m.Render
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
			// var (
			// 	x int16  = 0
			// 	y int16  = 0
			// 	w uint16 = 0
			// 	h uint16 = 0
			// )
			//
			// wid, err := CreateXSubWindow(conn, state.WID, x, y, w, h)
			// if err != nil {
			// 	return err
			// }
			//
			// token := sutureext.Add(super, Window{
			// 	wid:   wid,
			// 	main:  "",
			// 	sub:   "",
			// 	flags: []string{},
			// })
			//
			// state.Panes = append(state.Panes, StatePane{
			// 	UUID:    "",
			// 	WID:     wid,
			// 	Service: token,
			// 	X:       x,
			// 	Y:       y,
			// 	W:       w,
			// 	H:       h,
			// })

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
