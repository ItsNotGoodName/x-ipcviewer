package app

import (
	"math"
	"sync"

	"github.com/ItsNotGoodName/x-ipc-viewer/closer"
	"github.com/ItsNotGoodName/x-ipc-viewer/config"
	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/ItsNotGoodName/x-ipc-viewer/mpv"
	"github.com/ItsNotGoodName/x-ipc-viewer/xcursor"
	"github.com/ItsNotGoodName/x-ipc-viewer/xwm"
	"github.com/fsnotify/fsnotify"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/spf13/viper"
)

func Run(cfg *config.Config) error {
	x, err := xgb.NewConn()
	if err != nil {
		return err
	}
	defer x.Close()
	closer.Add(func() error {
		x.Close()
		return nil
	})

	if cfg.ConfigWatchExit {
		viper.WatchConfig()
		viper.OnConfigChange(func(in fsnotify.Event) {
			x.Close()
		})
	}

	// Cursor
	cursor, err := xcursor.CreateCursor(x, xcursor.LeftPtr)
	if err != nil {
		return err
	}

	// Layout
	var layout mosaic.Layout
	if cfg.Layout.IsAuto() {
		layout = mosaic.NewLayoutGridCount(len(cfg.Windows))
	} else {
		layout = mosaic.NewLayoutManual(cfg.LayoutManualWindows)
	}

	// Manager
	manager, err := xwm.NewManager(x, xproto.Setup(x).DefaultScreen(x), cursor, mosaic.New(layout))
	if err != nil {
		return err
	}
	defer manager.Release()

	// Create windows
	windows, err := createWindows(cfg, x, manager.WID(), layout)
	if err != nil {
		return err
	}

	// Add windows
	manager.AddWindows(x, windows)

	// Events
	xwm.HandleEvent(x, manager)

	return nil
}

func createWindows(cfg *config.Config, x *xgb.Conn, root xproto.Window, layout mosaic.Layout) ([]xwm.Window, error) {
	count := int(math.Min(float64(layout.Count()), float64(len(cfg.Windows))))

	windows := make([]xwm.Window, count)
	wg := sync.WaitGroup{}
	errC := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Create X window
			w, err := xwm.CreateXSubWindow(x, root)
			if err != nil {
				errC <- err
				return
			}

			// Crate player factory
			pf := mpv.NewPlayerFactory(cfg.Windows[i].Name, cfg.Windows[i].Flags, cfg.Player.GPU, cfg.Windows[i].LowLatency)

			// Create player
			p, err := pf(w)
			if err != nil {
				errC <- err
				return
			}
			p = xwm.NewPlayerCache(p)

			// Create window
			windows[i] = xwm.NewWindow(w, p, cfg.Windows[i].Main, cfg.Windows[i].Sub, cfg.Background)
		}(i)
	}
	wg.Wait()
	select {
	case err := <-errC:
		return nil, err
	default:
		return windows, nil
	}
}
