package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb/xproto"
)

var ErrPlayerClosed = errors.New("player closed")

type PlayerState string

var (
	PlayerStatePaused  PlayerState = "paused"
	PlayerStatePlaying PlayerState = "playing"
)

type (
	PlayerCommandState struct {
		State PlayerState
	}
	PlayerCommandVolume struct {
		Volume int
	}
	PlayerCommandLoad struct {
		File string
	}
)

func NewPlayer(ctx context.Context, wid xproto.Window, hwdec string) (Player, error) {
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
	if hwdec != "" {
		_ = m.SetOptionString("hwdec", hwdec)
	}
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
		closeC: make(chan struct{}),
		file:   "",
		volume: 0,
		state:  PlayerStatePaused,
	}

	go p.run(ctx, m)

	return p, nil
}

type Player struct {
	eventC chan any
	doneC  chan struct{}
	closeC chan struct{}
	file   string
	volume int
	state  PlayerState
}

func (p Player) Send(ctx context.Context, cmds ...any) error {
	for _, cmd := range cmds {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.doneC:
			return ErrPlayerClosed
		case p.eventC <- cmd:
			return nil
		}
	}
	return nil
}

func (p Player) Close(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.doneC:
		return nil
	case p.closeC <- struct{}{}:
		<-p.doneC
		return nil
	}
}

func (p Player) run(ctx context.Context, m *mpv.Mpv) {
	defer func() {
		m.TerminateDestroy()
		close(p.doneC)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.closeC:
			return
		case e := <-p.eventC:
			switch e := e.(type) {
			case PlayerCommandState:
				if e.State == p.state {
					continue
				}
				if e.State == PlayerStatePlaying {
					if err := m.Command([]string{"loadfile", p.file}); err != nil {
						slog.Error("Failed to play file", "error", err)
						continue
					}
				} else {
					if err := m.Command([]string{"stop"}); err != nil {
						slog.Error("Failed to stop", "error", err)
						continue
					}
				}

				p.state = e.State
			case PlayerCommandLoad:
				if e.File == p.file {
					continue
				}
				if err := m.Command([]string{"loadfile", e.File}); err != nil {
					slog.Error("Failed to load file", "error", err)
					continue
				}
				p.file = e.File
			case PlayerCommandVolume:
				if e.Volume == p.volume {
					continue
				}
				if err := m.SetProperty("volume", mpv.FormatInt64, int64(e.Volume)); err != nil {
					slog.Error("Failed to set volume", "error", err)
				}
				p.volume = e.Volume
			}
		}
	}
}

// func (p Player) handleEvents(m *mpv.Mpv, shutdownC chan struct{}) {
// 	go func() {
// 		for {
// 			select {
// 			case <-shutdownC:
// 				return
// 			default:
// 			}
//
// 			e := m.WaitEvent(10000)
// 			if e.Error != nil {
// 				slog.Error("Failed to listen for events", "error", e.Error)
// 				return
// 			}
//
// 			switch e.EventID {
// 			case mpv.EventPropertyChange:
// 				prop := e.Property()
// 				value := prop.Data.(int)
// 				fmt.Println("property:", prop.Name, value)
// 			case mpv.EventFileLoaded:
// 				p, err := m.GetProperty("media-title", mpv.FormatString)
// 				if err != nil {
// 					fmt.Println("error:", err)
// 				}
// 				fmt.Println("title:", p.(string))
// 			case mpv.EventLogMsg:
// 				msg := e.LogMessage()
// 				fmt.Println("message:", msg.Text)
// 			case mpv.EventStart:
// 				sf := e.StartFile()
// 				fmt.Println("start:", sf.EntryID)
// 			case mpv.EventEnd:
// 				ef := e.EndFile()
// 				fmt.Println("end:", ef.EntryID, ef.Reason)
// 				if ef.Reason == mpv.EndFileEOF {
// 					return
// 				} else if ef.Reason == mpv.EndFileError {
// 					fmt.Println("error:", ef.Error)
// 				}
// 			case mpv.EventShutdown:
// 				fmt.Println("shutdown:", e.EventID)
// 				return
// 			default:
// 				fmt.Println("event:", e.EventID)
// 			}
// 		}
// 	}()
// }
