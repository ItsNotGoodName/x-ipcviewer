package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb/xproto"
)

const MaxDelay = 0.5

const (
	DemuxerCacheIdle = "demuxer-cache-idle"
	// DemuxerCacheStart = "demuxer-start-time"
	DemuxerCacheTime = "demuxer-cache-time"
	TimeRemaining    = "time-remaining"
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
	_ = m.SetOptionString("profile", "low-latency")        // low latency for RTSP streams
	_ = m.SetOptionString("cache", "no")                   // get latest video for RTSP streams
	// _ = m.SetOptionString("demuxer-readahead-secs", "5")   // buffer a max of 5 seconds ahead

	// Custom options
	if hwdec != "" {
		_ = m.SetOptionString("hwdec", hwdec)
	}

	_ = m.RequestLogMessages("info")
	_ = m.ObserveProperty(0, DemuxerCacheTime, mpv.FormatDouble)
	// _ = m.ObserveProperty(0, DemuxerCacheStart, mpv.FormatDouble)
	_ = m.ObserveProperty(0, TimeRemaining, mpv.FormatDouble)

	if err := m.Initialize(); err != nil {
		return Player{}, err
	}

	p := Player{
		commandC: make(chan any),
		doneC:    make(chan struct{}),
		closeC:   make(chan struct{}),
	}

	go p.run(ctx, m)

	return p, nil
}

type Player struct {
	commandC chan any
	doneC    chan struct{}
	closeC   chan struct{}
}

func (p Player) Send(ctx context.Context, cmds ...any) error {
	for _, cmd := range cmds {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.doneC:
			return ErrPlayerClosed
		case p.commandC <- cmd:
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
	defer close(p.doneC)
	defer m.TerminateDestroy()

	// Tickers
	eventTicker := time.NewTicker(time.Second)
	defer eventTicker.Stop()

	watchTicker := time.NewTicker(time.Second)
	defer watchTicker.Stop()

	// Ping
	lastPing := time.Now()
	ping := func() {
		lastPing = time.Now()
	}

	// Signals
	state := NewSignal(PlayerStatePaused)
	file := NewSignal("")
	volume := NewSignal(0)
	speed := NewSignal(1.0)

	syncFile := func() {
		if file.V == "" {
			if err := m.Command([]string{"stop"}); err != nil {
				slog.Error("Failed to stop", "error", err)
			}
		} else {
			if err := m.Command([]string{"loadfile", file.V}); err != nil {
				slog.Error("Failed to play file", "error", err)
			}
		}
		ping()
	}
	file.AddEffect(syncFile)

	syncVolume := func() {
		if err := m.SetProperty("volume", mpv.FormatInt64, int64(volume.V)); err != nil {
			slog.Error("Failed to set volume", "error", err)
		}
	}
	volume.AddEffect(syncVolume)

	syncSpeed := func() {
		if err := m.SetProperty("speed", mpv.FormatDouble, speed.V); err != nil {
			slog.Error("Failed to set speed", "error", err)
		}
	}
	speed.AddEffect(syncSpeed)

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.closeC:
			return
		case <-watchTicker.C:
			if lastPing.Add(5 * time.Second).Before(time.Now()) {
				syncFile()
			}
		case <-eventTicker.C:
		eventLoop:
			for {
				e := m.WaitEvent(0)
				if e.Error != nil {
					slog.Error("Failed to listen for events", "error", e.Error)
					break
				}

				switch e.EventID {
				case mpv.EventNone:
					break eventLoop
				case mpv.EventPropertyChange:
					prop := e.Property()
					fmt.Println("property:", prop.Name, prop.Data)

					switch prop.Name {
					// case DemuxerCacheStart:
					case DemuxerCacheTime:
						ping()
					case TimeRemaining:
						if prop.Data.(float64) > MaxDelay {
							speed.SetValue(1.5)
						} else {
							speed.SetValue(1)
						}
					}
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
					if ef.Reason == mpv.EndFileError {
						fmt.Println("error:", ef.Error)
					}
				case mpv.EventShutdown:
					fmt.Println("shutdown:", e.EventID)
					break eventLoop
				default:
					fmt.Println("event:", e.EventID)
				}
			}
		case c := <-p.commandC:
			switch c := c.(type) {
			case PlayerCommandState:
				state.SetValue(c.State)
			case PlayerCommandLoad:
				file.SetValue(c.File)
			case PlayerCommandVolume:
				volume.SetValue(c.Volume)
			}
		}
	}
}
