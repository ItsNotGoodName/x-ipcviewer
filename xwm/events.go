package xwm

import (
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func HandleEvents(x *xgb.Conn, m *Manager) {
	// Start the main event loop.
	for {
		// WaitForEvent either returns an event or an error and never both.
		// If both are nil, then something went wrong and the loop should be
		// halted.
		//
		// An error can only be seen here as a response to an unchecked
		// request.
		ev, xerr := x.WaitForEvent()
		if ev == nil && xerr == nil {
			fmt.Println("Both event and error are nil. Exiting...")
			return
		}

		if ev != nil {
			fmt.Printf("Event: %s\n", ev)
		}
		if xerr != nil {
			fmt.Printf("Error: %s\n", xerr)
		}

		// This is how accepting events work:
		// The application checks what event we got and reacts to it
		// accordingly. All events are defined in the xproto subpackage.
		// To receive events, we have to first register it using
		// either xproto.CreateWindow or xproto.ChangeWindowAttributes.
		switch ev := ev.(type) {
		case xproto.ConfigureNotifyEvent:
			if m.RootChanged(ev.Width, ev.Height) {
				m.UpdateRoot(x, ev.Width, ev.Height)
			}
		case xproto.ButtonPressEvent:
			fmt.Printf("Button pressed: %d\n", ev.Detail)

			m.ButtonPress(x, ev)
		case xproto.KeyPressEvent:
			// See https://pkg.go.dev/github.com/jezek/xgb/xproto#KeyPressEvent
			// for documentation about a key press event.
			fmt.Printf("Key pressed: %d\n", ev.Detail)
			// The Detail value depends on the keyboard layout,
			// for QWERTY, q is #24.
			if ev.Detail == 24 {
				return // exit on q
			}

			m.KeyPress(ev)
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
			return
		}
	}
}
