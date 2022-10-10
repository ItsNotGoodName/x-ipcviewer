package xwm

import (
	"log"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type EventHandler interface {
	ConfigureNotify(x *xgb.Conn, ev xproto.ConfigureNotifyEvent)
	ButtonPress(x *xgb.Conn, ev xproto.ButtonPressEvent)
	KeyPress(x *xgb.Conn, ev xproto.KeyPressEvent)
}

func HandleEvent(x *xgb.Conn, eh EventHandler) {
	for {
		ev, err := x.WaitForEvent()
		if ev == nil && err == nil {
			log.Println("xwm.HandleEvent: exit: no event or error")
			return
		}

		if err != nil {
			log.Println("xwm.HandleEvent: error:", err)
		}

		switch ev := ev.(type) {
		case xproto.ConfigureNotifyEvent:
			log.Println("xwm.HandleEvent:", ev)

			eh.ConfigureNotify(x, ev)
		case xproto.ButtonPressEvent:
			log.Println("xwm.HandleEvent: button press event:", ev.Detail)

			eh.ButtonPress(x, ev)
		case xproto.KeyPressEvent:
			log.Println("xwm.HandleEvent: key press event:", ev.Detail)

			if ev.Detail == 24 { // q
				log.Println("xwm.HandleEvent: exit: quit key pressed")
				return
			}

			eh.KeyPress(x, ev)
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
			log.Println("xwm.HandleEvent: exit: destroy notify event")

			return
		default:
			log.Println("xwm.HandleEvent: unknown event:", ev)
		}
	}
}
