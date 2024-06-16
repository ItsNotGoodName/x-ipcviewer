package xwm

import (
	"context"
	"log/slog"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/config"
	"github.com/ItsNotGoodName/x-ipcviewer/internal/xcursor"
	"github.com/ItsNotGoodName/x-ipcviewer/pkg/sutureext"
	"github.com/google/uuid"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/thejerf/suture/v4"
)

func SetupState(conn *xgb.Conn, provider config.Provider) (State, error) {
	screen := xproto.Setup(conn).DefaultScreen(conn)

	cursor, err := xcursor.CreateCursor(conn, xcursor.LeftPtr)
	if err != nil {
		return State{}, err
	}

	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		return State{}, err
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
		return State{}, err
	}

	if err := xproto.MapWindowChecked(conn, wid).Check(); err != nil {
		return State{}, err
	}

	return State{
		WID:            wid,
		Width:          screen.WidthInPixels,
		Height:         screen.HeightInPixels,
		Panes:          []StatePane{},
		FullscreenUUID: "",
	}, nil
}

func HandleEvents(ctx context.Context, conn *xgb.Conn, state State, eventC chan any) error {
	super := suture.NewSimple("")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-eventC:
			if !ok {
				return nil
			}

			switch ev := ev.(type) {
			case xproto.ConfigureNotifyEvent:
				slog.Debug("ConfigureNotifyEvent:", "event", ev)

				state.UpdateWindow(ev.Width, ev.Height)
			case xproto.ButtonPressEvent:
				slog.Debug("ButtonPressEvent", "detail", ev.Detail)
			case xproto.KeyPressEvent:
				slog.Debug("KeyPressEvent", "detail", ev.Detail)

				if ev.Detail == 24 {
					slog.Debug("exit: quit key pressed")
					return nil
				}

				if ev.Detail == 65 {
					var (
						x int16  = 0
						y int16  = 0
						w uint16 = 0
						h uint16 = 0
					)

					wid, err := CreateXSubWindow(conn, state.WID, x, y, w, h)
					if err != nil {
						return err
					}

					token := sutureext.Add(super, Window{
						wid:   wid,
						main:  "",
						sub:   "",
						flags: []string{},
					})

					state.Panes = append(state.Panes, StatePane{
						UUID:    "",
						WID:     wid,
						Service: token,
						X:       x,
						Y:       y,
						W:       w,
						H:       h,
					})

					return nil
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
	slog := slog.With("func", "xwm.ReceiveEvents")

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

func CreateXSubWindow(conn *xgb.Conn, root xproto.Window, x, y int16, w, h uint16) (xproto.Window, error) {
	// Generate X window id
	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		return 0, err
	}

	// Create X window in root
	if err := xproto.CreateWindowChecked(conn, xproto.WindowClassCopyFromParent,
		wid, root,
		x, y, w, h, 0,
		xproto.WindowClassInputOutput, xproto.WindowClassCopyFromParent, 0, []uint32{}).Check(); err != nil {
		return 0, err
	}

	// Show X window
	if err = xproto.MapWindowChecked(conn, wid).Check(); err != nil {
		xproto.DestroyWindow(conn, wid)
		return 0, err
	}

	return wid, nil
}

func NormalizeConfig(provider config.Provider) error {
	return provider.UpdateConfig(func(cfg config.Config) (config.Config, error) {
		for i := range cfg.Streams {
			if cfg.Streams[i].UUID == "" {
				cfg.Streams[i].UUID = uuid.NewString()
			}
		}

		for i := range cfg.Views {
			if cfg.Views[i].UUID == "" {
				cfg.Views[i].UUID = uuid.NewString()
			}
		}

		return cfg, nil
	})
}
