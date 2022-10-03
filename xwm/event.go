package xwm

import (
	"fmt"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func HandleEvent(x *xgb.Conn, m *Manager) {
	eventC, errC := handleEvent(x)
	buttonPressC := make(chan xproto.ButtonPressEvent)
	singleButtonPressC, doubleButtonPressC := handleButtonPress(200, buttonPressC)

	for {
		// This is how accepting events work:
		// The application checks what event we got and reacts to it
		// accordingly. All events are defined in the xproto subpackage.
		// To receive events, we have to first register it using
		// either xproto.CreateWindow or xproto.ChangeWindowAttributes.
		select {
		case ev := <-singleButtonPressC:
			fmt.Printf("Single button pressed: %d\n", ev.Detail)

			m.ButtonPress(x, ev, false)
		case ev := <-doubleButtonPressC:
			fmt.Printf("Double button pressed: %d\n", ev.Detail)

			m.ButtonPress(x, ev, true)
		case _, ok := <-errC:
			if !ok {
				return
			}
		case ev, ok := <-eventC:
			if !ok {
				return
			}

			switch ev := ev.(type) {
			case xproto.ConfigureNotifyEvent:
				if m.ShouldUpdateRoot(ev.Width, ev.Height) {
					m.UpdateRoot(x, ev.Width, ev.Height)
				}
			case xproto.ButtonPressEvent:
				buttonPressC <- ev
			case xproto.KeyPressEvent:
				// See https://pkg.go.dev/github.com/jezek/xgb/xproto#KeyPressEvent
				// for documentation about a key press event.
				fmt.Printf("Key pressed: %d\n", ev.Detail)
				// The Detail value depends on the keyboard layout,
				// for QWERTY, q is #24.
				if ev.Detail == 24 {
					return // exit on q
				}

				m.KeyPress(x, ev)
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
}

func handleEvent(x *xgb.Conn) (<-chan xgb.Event, <-chan xgb.Error) {
	eventC, errC := make(chan xgb.Event), make(chan xgb.Error)
	go func() {
		defer close(eventC)
		defer close(errC)

		// Start the main event loop.
		for {
			// WaitForEvent either returns an event or an error and never both.
			// If both are nil, then something went wrong and the loop should be
			// halted.
			//
			// An error can only be seen here as a response to an unchecked
			// request.
			event, err := x.WaitForEvent()
			if event == nil && err == nil {
				fmt.Println("Both event and error are nil. Exiting...")
				return
			}

			if event != nil {
				fmt.Printf("Event: %s\n", event)
				eventC <- event
			}
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				errC <- err
			}
		}
	}()

	return eventC, errC
}

func handleButtonPress(timeout xproto.Timestamp, eventC <-chan xproto.ButtonPressEvent) (<-chan xproto.ButtonPressEvent, <-chan xproto.ButtonPressEvent) {
	singleC := make(chan xproto.ButtonPressEvent, 100)
	doubleC := make(chan xproto.ButtonPressEvent, 100)

	go func() {
		d := time.Duration(timeout) * time.Millisecond
		t := time.NewTimer(d)
		if !t.Stop() {
			<-t.C
		}
		drained := true
		lastEventC := make(chan xproto.ButtonPressEvent, 1)

		for {
			select {
			case <-t.C:
				// Single click
				select {
				case e := <-lastEventC:
					select {
					case singleC <- e:
					default:
					}
				default:
				}
				drained = true
			case event := <-eventC:
				select {
				case lastEvent := <-lastEventC:
					if event.Detail != lastEvent.Detail || (event.Time-lastEvent.Time) > timeout {
						// Single click
						select {
						case singleC <- lastEvent:
						default:
						}
						lastEventC <- event
						if !t.Stop() && !drained {
							<-t.C
							drained = true
						}
						t.Reset(d)
					} else {
						// Double click
						select {
						case doubleC <- event:
						default:
						}
						if !t.Stop() && !drained {
							<-t.C
							drained = true
						}
					}
				default:
					// Wait for next event or timeout
					lastEventC <- event
					if !t.Stop() && !drained {
						<-t.C
						drained = true
					}
					t.Reset(d)
				}
			}
		}
	}()

	return singleC, doubleC
}
