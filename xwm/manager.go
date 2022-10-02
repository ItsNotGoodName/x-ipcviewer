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
	mosaic mosaic.Mosaic

	fullscreen int
	lastWidth  uint16
	lastHeight uint16
	containers []Container
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
		wid:        wid,
		screen:     screen,
		lastWidth:  width,
		lastHeight: height,
		mosaic:     m,
		containers: []Container{},
		fullscreen: -1,
	}, nil
}

func (m *Manager) AddContainer(x *xgb.Conn, windowFactory WindowFactory, config ContainerConfig) error {
	// Generate window id
	wid, err := xproto.NewWindowId(x)
	if err != nil {
		return err
	}

	// Create window in root
	if xproto.CreateWindow(x, m.screen.RootDepth, wid, m.wid,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput, m.screen.RootVisual, 0, []uint32{}).Check(); err != nil {
		return err
	}

	// Create window
	window, err := windowFactory(wid)
	if err != nil {
		return err
	}

	// Create container
	container := Container{WID: wid, Window: window, MainStream: config.MainStream, SubStream: config.SubStream}
	m.containers = append(m.containers, container)

	// Update root
	m.UpdateRoot(x, m.lastWidth, m.lastHeight)

	// Show window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		return err
	}
	if err := window.Start(container.DefaultStream()); err != nil {
		return err
	}

	return err
}

func (m *Manager) RootChanged(width, height uint16) bool {
	return width != m.lastWidth || height != m.lastHeight
}

func (m *Manager) ToggleFullscreen(x *xgb.Conn, idx int) {
	if idx != -1 && idx == m.fullscreen {
		m.fullscreen = -1
		for _, container := range m.containers {
			if err := container.Window.Mute(true); err != nil {
				log.Printf("xwm.Manager.ToggleFullscreen: window %d: mute: %s\n", container.WID, err)
			}
			if err := container.Window.Start(container.DefaultStream()); err != nil {
				log.Printf("xwm.Manager.ToggleFullscreen: window %d: start: %s\n", container.WID, err)
			}
		}
	} else {
		m.fullscreen = idx

		for i, container := range m.containers {
			if i == idx {
				if err := xproto.ConfigureWindowChecked(x, container.WID, xproto.ConfigWindowStackMode, []uint32{0}).Check(); err != nil {
					log.Printf("xwm.Manager.ToggleFullscreen: window %d: stack: %s\n", container.WID, err)
				}
				if err := container.Window.Mute(false); err != nil {
					log.Printf("xwm.Manager.ToggleFullscreen: window %d: mute: %s\n", container.WID, err)
				}
				if err := container.Window.Start(container.DefaultStream()); err != nil {
					log.Printf("xwm.Manager.ToggleFullscreen: window %d: start: %s\n", container.WID, err)
				}
			} else {
				if err := container.Window.Stop(); err != nil {
					log.Printf("xwm.Manager.ToggleFullscreen: window %d: stop: %s\n", container.WID, err)
				}
			}
		}
	}

	m.UpdateRoot(x, m.lastWidth, m.lastHeight)
}

func (m *Manager) UpdateRoot(x *xgb.Conn, width, height uint16) {
	if m.fullscreen != -1 {
		if err := xproto.ConfigureWindowChecked(x, m.containers[m.fullscreen].WID, xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{uint32(0), uint32(0), uint32(width), uint32(height)}).Check(); err != nil {
			log.Printf("xwm.Manager.UpdateRoot: window %d: %s\n", m.containers[m.fullscreen].WID, err)
		}
	} else {
		mosaicWindows := m.mosaic.Windows(width, height)
		containersLength, mosaicWindowsLength := len(m.containers), len(mosaicWindows)
		for i := 0; i < containersLength && i < mosaicWindowsLength; i++ {
			container := m.containers[i]

			if err := xproto.ConfigureWindowChecked(x, container.WID, xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{uint32(mosaicWindows[i].X), uint32(mosaicWindows[i].Y), uint32(mosaicWindows[i].Width), uint32(mosaicWindows[i].Height)}).Check(); err != nil {
				log.Printf("xwm.Manager.UpdateRoot: window %d: %s\n", container.WID, err)
			}
		}
	}

	m.lastWidth, m.lastHeight = width, height
}

func (m *Manager) KeyPress(x *xgb.Conn, ev xproto.KeyPressEvent) {
	// Numbers 1 - 9
	if ev.Detail >= 10 && ev.Detail <= 18 {
		cLen := len(m.containers)
		idx := int(ev.Detail - 10)
		if idx < cLen {
			m.ToggleFullscreen(x, idx)
		}

		return
	}

	for _, container := range m.containers {
		if container.WID == ev.Child {
			container.Window.KeyPress(ev)
		}
	}
}

func (m *Manager) ButtonPress(x *xgb.Conn, ev xproto.ButtonPressEvent) {
	for i, container := range m.containers {
		if container.WID == ev.Child {
			if ev.Detail == 1 {
				m.ToggleFullscreen(x, i)
			} else {
				container.Window.ButtonPress(ev)
			}
			return
		}
	}
}

func (m *Manager) Release() {
	for _, container := range m.containers {
		container.Window.Release()
	}
}
