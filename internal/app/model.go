package app

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/config"
	"github.com/ItsNotGoodName/x-ipcviewer/internal/xwm"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Model struct {
	Store config.Store

	RootWID          xproto.Window
	RootWidth        uint16
	RootHeight       uint16
	StreamFullscreen string
	StreamSelected   string
	Streams          []ModelStream
	View             string
	Views            []ModelView
}

type ModelStream struct {
	UUID   string
	WID    xproto.Window
	Main   string
	Sub    string
	Player Player
}

type ModelView struct {
	X int16
	Y int16
	W uint16
	H uint16
}

func (m Model) Init(ctx context.Context, conn *xgb.Conn) (xwm.Model, xwm.Cmd) {
	window, err := xwm.CreateWindow(conn)
	if err != nil {
		panic(err)
	}

	m.RootWID = window.WID
	m.RootWidth = window.Width
	m.RootHeight = window.Height

	config, err := m.Store.GetConfig()
	if err != nil {
		panic(err)
	}

	m.View = config.View

	return m, nil
}

func (m Model) Update(ctx context.Context, conn *xgb.Conn, msg xwm.Msg) (xwm.Model, xwm.Cmd) {
	switch ev := msg.(type) {
	case xproto.ConfigureNotifyEvent:
		slog.Debug("ConfigureNotifyEvent:", "event", ev.String())

		if ev.Window == m.RootWID {
			m.RootWidth = ev.Width
			m.RootHeight = ev.Height
		}

		return m, nil
	case xproto.ButtonPressEvent:
		slog.Debug("ButtonPressEvent", "detail", ev.String())

		switch ev.Detail {
		case xproto.ButtonIndex1: // Left click
			idx := slices.IndexFunc(m.Streams, func(p ModelStream) bool { return ev.Child == p.WID })
			if idx == -1 {
				return m, nil
			}

			return m.fullscreen(ctx, m.Streams[idx].UUID), nil
		case xproto.ButtonIndex3: // Right click
			return m.fullscreen(ctx, ""), nil
		}

		return m, nil
	case xproto.KeyPressEvent:
		slog.Debug("KeyPressEvent", "detail", ev.String())

		switch ev.Detail {
		case 24: // q
			slog.Debug("exit: quit key pressed")
			return m.Close(conn), xwm.Quit
		case 113: // <left>
			if len(m.Streams) == 0 {
				return m, nil
			}

			idx := slices.IndexFunc(m.Streams, func(p ModelStream) bool { return p.UUID == m.StreamFullscreen })
			if idx == -1 {
				return m.fullscreen(ctx, m.Streams[len(m.Streams)-1].UUID), nil
			}

			return m.fullscreen(ctx, m.Streams[(len(m.Streams)+idx-1)%len(m.Streams)].UUID), nil
		case 114: // <right>
			if len(m.Streams) == 0 {
				return m, nil
			}

			idx := slices.IndexFunc(m.Streams, func(p ModelStream) bool { return p.UUID == m.StreamFullscreen })
			if idx == -1 {
				return m.fullscreen(ctx, m.Streams[0].UUID), nil
			}

			return m.fullscreen(ctx, m.Streams[(idx+1)%len(m.Streams)].UUID), nil
		case 111: // <up>
			return m, nil
		case 116: // <down>
			return m, nil
		case 166: // <back>
			return m.fullscreen(ctx, ""), nil
		case 65: // <space>
			config, err := m.Store.GetConfig()
			if err != nil {
				return m, xwm.Error(err)
			}

			for i, stream := range config.Streams {
				wid, err := xwm.CreateSubWindow(conn, m.RootWID)
				if err != nil {
					return m, xwm.Error(err)
				}

				player, err := NewPlayer(ctx, wid)
				if err != nil {
					xwm.DestroySubWindow(conn, wid)
					return m, xwm.Error(err)
				}

				m.Streams = append(m.Streams, ModelStream{
					UUID:   stream.UUID,
					WID:    wid,
					Main:   stream.Main,
					Sub:    stream.Sub,
					Player: player,
				})

				player.Send(ctx, PlayerCommandLoadFile{
					File: stream.Sub,
				})

				if i == 0 {
					m.StreamSelected = stream.UUID
				} else {
					player.Send(ctx, PlayerCommandVolume{
						Volume: 0,
					})
				}
			}

			return m, nil
		default:
			return m, nil
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

		return m.Close(conn), xwm.Quit
	default:
		slog.Debug("unknown event", "event", ev)
		return m, nil
	}
}

func (m Model) Render(ctx context.Context, conn *xgb.Conn) error {
	idx := slices.IndexFunc(m.Streams, func(p ModelStream) bool { return p.UUID == m.StreamFullscreen })

	if idx == -1 {
		if m.View != "grid" {
			return fmt.Errorf("view %s not supported", m.View)
		}

		layout := NewLayoutGrid(m.RootWidth, m.RootHeight, len(m.Streams))
		for i := range m.Streams {
			x, y, w, h := layout.Pane(i)

			err := xproto.ConfigureWindowChecked(conn, m.Streams[i].WID,
				xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
				[]uint32{uint32(x), uint32(y), uint32(w), uint32(h)}).
				Check()
			if err != nil {
				return err
			}
		}

		return nil
	} else {
		x, y, w, h := int16(0), int16(0), m.RootWidth, m.RootHeight

		err := xproto.ConfigureWindowChecked(conn, m.Streams[idx].WID,
			xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight|xproto.ConfigWindowStackMode,
			[]uint32{uint32(x), uint32(y), uint32(w), uint32(h), 0}).
			Check()
		if err != nil {
			fmt.Println(err)
			return err
		}

		return nil
	}
}

func (m Model) Close(conn *xgb.Conn) Model {
	return m
}

func (m Model) fullscreen(ctx context.Context, uuid string) Model {
	m.StreamFullscreen = uuid

	for _, p := range m.Streams {
		if p.UUID == m.StreamFullscreen {
			p.Player.Send(ctx, PlayerCommandLoadFile{
				File: p.Main,
			})
		} else {
			p.Player.Send(ctx, PlayerCommandLoadFile{
				File: p.Sub,
			})
		}
	}

	return m
}
