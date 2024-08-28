package app

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/config"
	"github.com/ItsNotGoodName/x-ipcviewer/internal/xplayer"
	"github.com/ItsNotGoodName/x-ipcviewer/internal/xwm"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Model struct {
	Store config.Store

	LastLeftClick    time.Time
	RootWID          xproto.Window
	RootWidth        uint16
	RootHeight       uint16
	StreamGPU        string
	StreamFullscreen string
	StreamSelected   string
	Streams          []ModelStream
}

type ModelStream struct {
	UUID   string
	WID    xproto.Window
	Main   string
	Sub    string
	Player xplayer.Player
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

	m.StreamGPU = config.GPU

	return m.syncStreams(ctx, conn, config.Streams)
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
			var doubleClick bool
			m.LastLeftClick, doubleClick = checkClick(m.LastLeftClick)

			idx := slices.IndexFunc(m.Streams, func(p ModelStream) bool { return ev.Child == p.WID })
			if idx == -1 {
				return m, nil
			}

			if doubleClick {
				if m.StreamFullscreen == "" {
					m.StreamFullscreen = m.Streams[idx].UUID
				} else {
					m.StreamFullscreen = ""
				}
			} else {
				m.StreamSelected = m.Streams[idx].UUID
			}

			return m, nil
		case xproto.ButtonIndex3: // Right click
			m.StreamFullscreen = ""
			return m, nil
		default:
			return m, nil
		}
	case xproto.KeyPressEvent:
		slog.Debug("KeyPressEvent", "detail", ev.String())

		switch ev.Detail {
		case 24: // q
			slog.Debug("exit: quit key pressed")

			return m.Close(ctx, conn), xwm.Quit
		case 113: // <left>
			if len(m.Streams) == 0 {
				return m, nil
			}

			idx := slices.IndexFunc(m.Streams, func(p ModelStream) bool { return p.UUID == m.StreamFullscreen })
			if idx == -1 {
				m.StreamFullscreen = m.Streams[len(m.Streams)-1].UUID
			} else {
				m.StreamFullscreen = m.Streams[(len(m.Streams)+idx-1)%len(m.Streams)].UUID
			}

			return m, nil
		case 114: // <right>
			if len(m.Streams) == 0 {
				return m, nil
			}

			idx := slices.IndexFunc(m.Streams, func(p ModelStream) bool { return p.UUID == m.StreamFullscreen })
			if idx == -1 {
				m.StreamFullscreen = m.Streams[0].UUID
			} else {
				m.StreamFullscreen = m.Streams[(idx+1)%len(m.Streams)].UUID
			}

			return m, nil
		case 111: // <up>
			return m, nil
		case 116: // <down>
			return m, nil
		case 166: // <back>
			m.StreamFullscreen = ""
			return m, nil
		case 65, 135: // <space>
			config, err := m.Store.GetConfig()
			if err != nil {
				return m, xwm.Error(err)
			}

			return m.syncStreams(ctx, conn, config.Streams)
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

		return m.Close(ctx, conn), xwm.Quit
	default:
		slog.Debug("unknown event", "event", ev)
		return m, nil
	}
}

func (m Model) Render(ctx context.Context, conn *xgb.Conn) error {
	xproto.MapWindow(conn, m.RootWID)

	// File
	for _, s := range m.Streams {
		if s.UUID == m.StreamFullscreen {
			err := s.Player.Send(ctx, xplayer.CommandLoad{
				File: s.Main,
			})
			if err != nil {
				return err
			}
		} else {
			err := s.Player.Send(ctx, xplayer.CommandLoad{
				File: s.Sub,
			})
			if err != nil {
				return err
			}
		}
	}

	// Volume
	for _, s := range m.Streams {
		if s.UUID == m.StreamSelected {
			err := s.Player.Send(ctx, xplayer.CommandVolume{
				Volume: 100,
			})
			if err != nil {
				return err
			}
		} else {
			err := s.Player.Send(ctx, xplayer.CommandVolume{
				Volume: 0,
			})
			if err != nil {
				return err
			}
		}
	}

	// Layout
	if slices.ContainsFunc(m.Streams, func(p ModelStream) bool { return p.UUID == m.StreamFullscreen }) {
		// Fullscreen
		for _, s := range m.Streams {
			if s.UUID == m.StreamFullscreen {
				// Configure fullscreen
				err := xproto.ConfigureWindowChecked(conn, s.WID,
					xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight|xproto.ConfigWindowStackMode,
					[]uint32{uint32(0), uint32(0), uint32(m.RootWidth), uint32(m.RootHeight), 0}).
					Check()
				if err != nil {
					return err
				}

				// Show window
				if err = xproto.MapWindowChecked(conn, s.WID).Check(); err != nil {
					return err
				}

				// Play
				err = s.Player.Send(ctx, xplayer.CommandPlay{})
				if err != nil {
					return err
				}
			} else {
				xproto.UnmapWindow(conn, s.WID)
			}
		}
	} else {
		// Grid
		layout := NewLayoutGrid(m.RootWidth, m.RootHeight, len(m.Streams))
		for i := range m.Streams {
			x, y, w, h := layout.Pane(i)

			// Configure pane
			err := xproto.ConfigureWindowChecked(conn, m.Streams[i].WID,
				xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
				[]uint32{uint32(x), uint32(y), uint32(w), uint32(h)}).
				Check()
			if err != nil {
				return err
			}

			// Show window
			if err = xproto.MapWindowChecked(conn, m.Streams[i].WID).Check(); err != nil {
				return err
			}

			// Play
			err = m.Streams[i].Player.Send(ctx, xplayer.CommandPlay{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m Model) syncStreams(ctx context.Context, conn *xgb.Conn, cfgStreams []config.Stream) (xwm.Model, xwm.Cmd) {
	var newStreams []ModelStream
	for _, cfgStream := range cfgStreams {
		idx := slices.IndexFunc(m.Streams, func(s ModelStream) bool { return s.UUID == cfgStream.UUID })
		if idx == -1 {
			// Create
			wid, err := xwm.CreateSubWindow(conn, m.RootWID, 0, 0, m.RootWidth, m.RootHeight)
			if err != nil {
				return m, xwm.Error(err)
			}

			player, err := xplayer.NewPlayer(ctx, cfgStream.UUID, wid, m.StreamGPU)
			if err != nil {
				return m, xwm.Error(err)
			}

			newStreams = append(newStreams, ModelStream{
				UUID:   cfgStream.UUID,
				WID:    wid,
				Main:   cfgStream.Main,
				Sub:    cfgStream.Sub,
				Player: player,
			})
		} else {
			// Update
			stream := m.Streams[idx]
			stream.Main = cfgStream.Main
			stream.Sub = cfgStream.Sub
			newStreams = append(newStreams, stream)
		}
	}

	// Delete
	for _, stream := range m.Streams {
		if !slices.ContainsFunc(cfgStreams, func(s config.Stream) bool { return s.UUID == stream.UUID }) {
			closeModelStream(ctx, conn, stream)
		}
	}

	m.Streams = newStreams

	return m, nil
}

func (m Model) Close(ctx context.Context, conn *xgb.Conn) Model {
	for _, stream := range m.Streams {
		closeModelStream(ctx, conn, stream)
	}
	m.Streams = []ModelStream{}
	return m
}

func closeModelStream(ctx context.Context, conn *xgb.Conn, stream ModelStream) {
	stream.Player.Close(ctx)
	xproto.UnmapWindow(conn, stream.WID)
	xwm.DestroySubWindow(conn, stream.WID)
}

func checkClick(lastClick time.Time) (now time.Time, doubleClick bool) {
	now = time.Now()
	if now.Sub(lastClick) < 500*time.Millisecond {
		return time.Time{}, true
	} else {
		return now, false
	}
}
