package xwm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ItsNotGoodName/x-ipcviewer/xcursor"
	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type State struct {
	Conn           *xgb.Conn
	WID            xproto.Window
	Width          uint16
	Height         uint16
	Panes          []Pane
	FullscreenUUID string
}

type Pane struct {
	UUID string
	WID  int
	Y    int
	X    int
	W    int
	H    int
}

func Start(ctx context.Context) error {
	// X11 connection
	conn, err := xgb.NewConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	// X11 setup
	screen := xproto.Setup(conn).DefaultScreen(conn)

	cursor, err := xcursor.CreateCursor(conn, xcursor.LeftPtr)
	if err != nil {
		return err
	}

	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		return err
	}

	if err := xproto.CreateWindowChecked(conn, screen.RootDepth,
		wid, screen.Root,
		0, 0, screen.HeightInPixels, screen.HeightInPixels, 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask|xproto.CwCursor, // 1, 2, 3
		[]uint32{
			0, // 1
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskButtonPress, // 2
			uint32(cursor), // 3
		}).Check(); err != nil {
		return err
	}

	if err := xproto.MapWindowChecked(conn, wid).Check(); err != nil {
		return err
	}

	eventC := make(chan any)
	go ReceiveEvents(ctx, conn, eventC)

	quitC := make(chan struct{}, 1)

	state := State{
		WID:            wid,
		Width:          screen.WidthInPixels,
		Height:         screen.HeightInPixels,
		Panes:          []Pane{},
		FullscreenUUID: "",
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-quitC:
			return nil
		case ev, ok := <-eventC:
			if !ok {
				return nil
			}

			switch ev := ev.(type) {
			case xproto.ConfigureNotifyEvent:
				slog.Debug("ConfigureNotifyEvent:", "event", ev)

				state.Width = ev.Width
				state.Height = ev.Height
			case xproto.ButtonPressEvent:
				slog.Debug("ButtonPressEvent", "detail", ev.Detail)
			case xproto.KeyPressEvent:
				slog.Debug("KeyPressEvent", "detail", ev.Detail)

				if ev.Detail == 24 {
					slog.Debug("exit: quit key pressed")
					return nil
				}

				if ev.Detail == 65 {
					wid, err := xproto.NewWindowId(conn)
					if err != nil {
						return err
					}

					err = xproto.CreateWindowChecked(conn, xproto.WindowClassCopyFromParent,
						wid, state.WID,
						0, 0, 100, 100, 0,
						xproto.WindowClassInputOutput, xproto.WindowClassCopyFromParent, 0, []uint32{}).Check()
					if err != nil {
						return err
					}

					if err := xproto.MapWindowChecked(conn, wid).Check(); err != nil {
						return err
					}

					m := NewMPV(wid)
					_ = m.RequestLogMessages("info")
					_ = m.ObserveProperty(0, "pause", mpv.FormatFlag)

					_ = m.SetPropertyString("input-default-bindings", "yes")
					_ = m.SetOptionString("input-vo-keyboard", "yes")
					_ = m.SetOption("input-cursor", mpv.FormatFlag, true)
					_ = m.SetOption("osc", mpv.FormatFlag, true)
					_ = m.SetOption("force-window", mpv.FormatFlag, true)
					_ = m.SetOption("idle", mpv.FormatFlag, true)
					_ = m.SetOptionString("loop-file", "inf")

					if err = m.Initialize(); err != nil {
						return err
					}

					err = m.Command([]string{"loadfile", os.Args[1]})
					if err != nil {
						return err
					}

					go func() {
						for {
							e := m.WaitEvent(10000)

							switch e.EventID {
							case mpv.EventPropertyChange:
								prop := e.Property()
								value := prop.Data.(int)
								fmt.Println("property:", prop.Name, value)
							case mpv.EventFileLoaded:
								p, err := m.GetProperty("media-title", mpv.FormatString)
								if err != nil {
									fmt.Println("error:", err)
								}
								fmt.Println("title:", p.(string))
							case mpv.EventLogMsg:
								msg := e.LogMessage()
								fmt.Println("message:", msg.Text)
							case mpv.EventStart:
								sf := e.StartFile()
								fmt.Println("start:", sf.EntryID)
							case mpv.EventEnd:
								ef := e.EndFile()
								fmt.Println("end:", ef.EntryID, ef.Reason)
								if ef.Reason == mpv.EndFileEOF {
									return
								} else if ef.Reason == mpv.EndFileError {
									fmt.Println("error:", ef.Error)
								}
							case mpv.EventShutdown:
								fmt.Println("shutdown:", e.EventID)
								select {
								case quitC <- struct{}{}:
								default:
								}
								return
							default:
								fmt.Println("event:", e.EventID)
							}

							if e.Error != nil {
								fmt.Println("error:", e.Error)
							}
						}
					}()

				}
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

				return nil
			default:
				slog.Debug("unknown event", "event", ev)
			}
		}
	}
}

func ReceiveEvents(ctx context.Context, conn *xgb.Conn, eventC chan<- any) {
	defer close(eventC)
	slog := slog.With("func", "xwm.EventHandler.Serve")

	for {
		ev, err := conn.WaitForEvent()
		if ev == nil && err == nil {
			slog.Debug("exit: no event or error")
			return
		}

		if err != nil {
			slog.Error("failed to read event", "error", err)
			return
		}

		select {
		case <-ctx.Done():
			return
		case eventC <- ev:
		}
	}
}
