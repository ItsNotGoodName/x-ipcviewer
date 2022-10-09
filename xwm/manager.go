package xwm

import (
	"log"

	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// Manager is NOT concurrent safe.
type Manager struct {
	wid    xproto.Window
	screen *xproto.ScreenInfo
	mosaic mosaic.Mosaic

	lastButtonPressEv xproto.ButtonPressEvent
	fullscreenWid     xproto.Window
	width             uint16
	height            uint16
	windows           []Window
}

func NewManager(x *xgb.Conn, screen *xproto.ScreenInfo, m mosaic.Mosaic) (*Manager, error) {
	width, height := screen.WidthInPixels, screen.HeightInPixels

	// Generate x window id
	wid, err := xproto.NewWindowId(x)
	if err != nil {
		return nil, err
	}

	// Create main x window
	// Set x window background to black and listen for resize, key presses, and button presses
	if err := xproto.CreateWindowChecked(x, screen.RootDepth,
		wid, screen.Root,
		0, 0, width, height, 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			000000000,
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskButtonPress,
		}).Check(); err != nil {
		return nil, err
	}

	// Show x window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		xproto.DestroyWindow(x, wid)
		return nil, err
	}

	return &Manager{
		wid:           wid,
		screen:        screen,
		width:         width,
		height:        height,
		mosaic:        m,
		windows:       []Window{},
		fullscreenWid: 0,
	}, nil
}

func (m *Manager) AddWindows(x *xgb.Conn, windows []Window) {
	m.windows = append(m.windows, windows...)

	for i := range m.windows {
		m.windows[i].Show(false, false)
	}

	m.Update(x)
}

func (m *Manager) ToggleFullscreen(x *xgb.Conn, wid xproto.Window) {
	if wid == 0 || wid == m.fullscreenWid {
		// Normal
		m.fullscreenWid = 0

		for _, window := range m.windows {
			window.Show(false, false)
		}
	} else {
		// Fullscreen
		m.fullscreenWid = wid

		for _, window := range m.windows {
			if window.wid == wid {
				// Move fullscreen window to top of stack
				if err := xproto.ConfigureWindowChecked(x, window.wid, xproto.ConfigWindowStackMode, []uint32{0}).Check(); err != nil {
					log.Printf("xwm.Manager.ToggleFullscreen: window %d: stack: %s\n", window.wid, err)
				}
				window.Show(true, true)
			} else {
				window.Hide()
			}
		}
	}

	m.Update(x)
}

// Update root and children window's x, y, width, and height.
func (m *Manager) Update(x *xgb.Conn) {
	if m.fullscreenWid != 0 {
		// Fullscreen
		if err := xproto.ConfigureWindowChecked(x, m.fullscreenWid, xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{uint32(0), uint32(0), uint32(m.width), uint32(m.height)}).Check(); err != nil {
			log.Printf("xwm.Manager.UpdateRoot: window %d: %s\n", m.fullscreenWid, err)
		}
	} else {
		// Normal
		mosaicWindows := m.mosaic.Windows(m.width, m.height)
		windowsLength, mosaicWindowsLength := len(m.windows), len(mosaicWindows)
		for i := 0; i < windowsLength && i < mosaicWindowsLength; i++ {
			window := m.windows[i]

			if err := xproto.ConfigureWindowChecked(x, window.wid, xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{uint32(mosaicWindows[i].X), uint32(mosaicWindows[i].Y), uint32(mosaicWindows[i].Width), uint32(mosaicWindows[i].Height)}).Check(); err != nil {
				log.Printf("xwm.Manager.UpdateRoot: window %d: %s\n", window.wid, err)
			}
		}
	}
}

func (m *Manager) Release() {
	for _, window := range m.windows {
		window.Release()
	}
}

func (m *Manager) ConfigureNotify(x *xgb.Conn, ev xproto.ConfigureNotifyEvent) {
	if ev.Width != m.width || ev.Height != m.height {
		m.width = ev.Width
		m.height = ev.Height
		m.Update(x)
	}
}

func (m *Manager) KeyPress(x *xgb.Conn, ev xproto.KeyPressEvent) {
	// Keypad 1 - 9
	if ev.Detail >= 10 && ev.Detail <= 18 {
		cLen := len(m.windows)
		idx := int(ev.Detail - 10)
		if idx < cLen {
			m.ToggleFullscreen(x, m.windows[idx].wid)
		}

		return
	}
}

func (m *Manager) ButtonPress(x *xgb.Conn, ev xproto.ButtonPressEvent) {
	m.buttonPress(x, ev, (ev.Detail == m.lastButtonPressEv.Detail && (ev.Time-m.lastButtonPressEv.Time) < 500))
	m.lastButtonPressEv = ev
}

func (m *Manager) buttonPress(x *xgb.Conn, ev xproto.ButtonPressEvent, double bool) {
	if ev.Detail == 1 {
		if double {
			m.ToggleFullscreen(x, ev.Child)
		}
	}
}

func (m *Manager) WID() xproto.Window {
	return m.wid
}
