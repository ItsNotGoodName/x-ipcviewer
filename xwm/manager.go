package xwm

import (
	"log"

	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Manager struct {
	wid    xproto.Window
	screen *xproto.ScreenInfo

	lastWidth  uint16
	lastHeight uint16
	mosaic     mosaic.Mosaic
	windows    []Window
}

func NewManager(x *xgb.Conn, screen *xproto.ScreenInfo, m mosaic.Mosaic) (*Manager, error) {
	width, height := screen.WidthInPixels, screen.HeightInPixels

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

	// Set background to white and listen for resize, key presses, and key releases
	xproto.ChangeWindowAttributesChecked(x, wid,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			0xffffffff,
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease,
		})

	manager := Manager{
		wid:        wid,
		screen:     screen,
		lastWidth:  width,
		lastHeight: height,
		mosaic:     m,
		windows:    []Window{},
	}

	manager.UpdateRoot(x, width, height)

	// Show window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		return nil, err
	}

	return &manager, nil
}

func (m *Manager) AddWindow(x *xgb.Conn) (Window, error) {
	wid, err := xproto.NewWindowId(x)
	if err != nil {
		return Window{}, err
	}

	window, err := NewWindow(wid)
	if err != nil {
		return Window{}, err
	}

	m.windows = append(m.windows, window)

	// Create window in root
	if xproto.CreateWindow(x, m.screen.RootDepth, wid, m.wid,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput, m.screen.RootVisual, 0, []uint32{}).Check(); err != nil {
		return Window{}, err
	}

	m.UpdateRoot(x, m.lastWidth, m.lastHeight)

	// Show window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		return Window{}, err
	}

	return window, err
}

func (m *Manager) RootChanged(width, height uint16) bool {
	return width != m.lastWidth || height != m.lastHeight
}

func (m *Manager) UpdateRoot(x *xgb.Conn, width, height uint16) {
	mosaicWindows := m.mosaic.Windows(width, height)
	windowsLength, mosaicWindowsLength := len(m.windows), len(mosaicWindows)
	for i := 0; i < windowsLength && i < mosaicWindowsLength; i++ {
		window := m.windows[i]

		if err := xproto.ConfigureWindowChecked(x, window.WID, xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{uint32(mosaicWindows[i].X), uint32(mosaicWindows[i].Y), uint32(mosaicWindows[i].Width), uint32(mosaicWindows[i].Height)}).Check(); err != nil {
			log.Printf("xwm.Manager.UpdateRoot: window %d: %s\n", window, err)
		}
	}

	m.lastWidth, m.lastHeight = width, height
}

func (m *Manager) HandleKeyPressEvent(ev xproto.KeyPressEvent) {
	for _, window := range m.windows {
		if window.WID == ev.Child {
			window.KeyPressEventC <- ev
		}
	}
}
