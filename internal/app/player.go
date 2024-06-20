package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb/xproto"
)

var ErrPlayerClosed = errors.New("player closed")

type (
	PlayerCommandPlay   struct{}
	PlayerCommandPause  struct{}
	PlayerCommandVolume struct {
		Volume int
	}
	PlayerCommandLoadFile struct {
		File string
	}
	PlayerCommandClose struct{}
)

func NewPlayer(ctx context.Context, wid xproto.Window) (Player, error) {
	m := mpv.New()

	// Base options
	_ = m.SetOption("wid", mpv.FormatInt64, int64(wid))    // bind to x window
	_ = m.SetOptionString("input-vo-keyboard", "no")       // passthrough keyboard input to sub x window
	_ = m.SetOption("input-cursor", mpv.FormatFlag, false) // passthrough mouse input to sub x window
	_ = m.SetOption("osc", mpv.FormatFlag, false)          // don't render on screen ui
	_ = m.SetOption("force-window", mpv.FormatFlag, true)  // render empty video when no file-loaded
	_ = m.SetOption("idle", mpv.FormatFlag, true)          // keep window open when no file-loaded
	_ = m.SetOptionString("loop-file", "inf")              // loop video

	// Custom options
	_ = m.SetOptionString("hwdec", "vaapi")
	_ = m.SetOptionString("profile", "low-latency")
	_ = m.SetOption("cache", mpv.FormatFlag, false)

	_ = m.RequestLogMessages("info")
	// _ = m.ObserveProperty(0, "pause", mpv.FormatFlag)

	if err := m.Initialize(); err != nil {
		return Player{}, err
	}

	p := Player{
		eventC: make(chan any),
		doneC:  make(chan struct{}),
	}

	go p.run(ctx, m)

	return p, nil
}

type Player struct {
	eventC chan any
	doneC  chan struct{}
}

func (p Player) Send(ctx context.Context, cmd any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.doneC:
		return ErrPlayerClosed
	case p.eventC <- cmd:
		return nil
	}
}

func (p Player) run(ctx context.Context, m *mpv.Mpv) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer close(p.doneC)

	p.handleEvents(m)

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-p.eventC:
			switch e := e.(type) {
			case PlayerCommandClose:
				return
			case PlayerCommandLoadFile:
				if err := m.Command([]string{"loadfile", e.File}); err != nil {
					slog.Error("Failed to load file", "error", err)
				}
			case PlayerCommandVolume:
				if err := m.SetProperty("volume", mpv.FormatInt64, int64(e.Volume)); err != nil {
					slog.Error("Failed to mute", "error", err)
				}
			}
		}
	}
}

func (p Player) handleEvents(m *mpv.Mpv) {
	go func() {
		for {
			e := m.WaitEvent(10000)
			if e.Error != nil {
				slog.Error("Failed to listen for events", "error", e.Error)
				return
			}

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
				return
			default:
				fmt.Println("event:", e.EventID)
			}
		}
	}()
}
