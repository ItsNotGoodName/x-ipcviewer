package xwm

import (
	"log"

	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Manager struct {
	WID    xproto.Window
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

	// Generate window id
	wid, err := xproto.NewWindowId(x)
	if err != nil {
		return nil, err
	}

	// Create main window
	if err := xproto.CreateWindowChecked(x, screen.RootDepth, wid, screen.Root,
		0, 0, width, height, 0,
		xproto.WindowClassInputOutput, screen.RootVisual, 0, []uint32{}).Check(); err != nil {
		return nil, err
	}

	// Set background to black and listen for resize, key presses, and key releases
	xproto.ChangeWindowAttributesChecked(x, wid,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			0x00000000,
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskButtonPress,
		})

	// Show window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		return nil, err
	}

	return &Manager{
		WID:           wid,
		screen:        screen,
		width:         width,
		height:        height,
		mosaic:        m,
		windows:       []Window{},
		fullscreenWid: 0,
	}, nil
}

func (m *Manager) AddWindow(x *xgb.Conn, factory PlayerFactory, config WindowConfig) error {
	// Generate window id
	wid, err := xproto.NewWindowId(x)
	if err != nil {
		return err
	}

	// Create window in root
	if xproto.CreateWindow(x, m.screen.RootDepth, wid, m.WID,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput, m.screen.RootVisual, 0, []uint32{}).Check(); err != nil {
		return err
	}

	// Create player
	player, err := factory(wid)
	if err != nil {
		return err
	}
	player = NewPlayerCache(player)

	// Create window
	window := NewWindow(wid, player, config)
	m.windows = append(m.windows, window)

	m.Update(x)

	// Show window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		return err
	}
	if err := window.Show(false, false); err != nil {
		return err
	}

	return err
}

func (m *Manager) ToggleFullscreen(x *xgb.Conn, wid xproto.Window) {
	if wid == 0 || wid == m.fullscreenWid {
		// Normal
		m.fullscreenWid = 0

		for _, window := range m.windows {
			if err := window.Show(false, false); err != nil {
				log.Printf("xwm.Manager.ToggleFullscreen: window %d: show: %s\n", window.wid, err)
			}
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
				if err := window.Show(true, true); err != nil {
					log.Printf("xwm.Manager.ToggleFullscreen: window %d: show: %s\n", window.wid, err)
				}
			} else {
				if err := window.Hide(); err != nil {
					log.Printf("xwm.Manager.ToggleFullscreen: window %d: hide: %s\n", window.wid, err)
				}
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
