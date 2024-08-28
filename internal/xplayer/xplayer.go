package xplayer

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb/xproto"
)

const MaxPlaybackDelay = 0.5

// https://mpv.io/manual/master/#property-list
const (
	PropertyDemuxerCacheTime = "demuxer-cache-time"
	PropertyTimeRemaining    = "time-remaining"
)

var ErrPlayerClosed = errors.New("player closed")

type (
	CommandLoad struct {
		File string
	}
	CommandPlayback struct {
		Playing bool
	}
	CommandVolume struct {
		Volume int
	}
)

func NewPlayer(ctx context.Context, id string, wid xproto.Window, hwdec string) (Player, error) {
	m := mpv.New()

	// Base options
	_ = m.SetOption("wid", mpv.FormatInt64, int64(wid))    // bind to x window
	_ = m.SetOptionString("input-vo-keyboard", "no")       // passthrough keyboard input to sub x window
	_ = m.SetOption("input-cursor", mpv.FormatFlag, false) // passthrough mouse input to sub x window
	_ = m.SetOption("osc", mpv.FormatFlag, false)          // don't render on screen ui
	_ = m.SetOption("force-window", mpv.FormatFlag, true)  // render empty video when no file-loaded
	_ = m.SetOption("idle", mpv.FormatFlag, true)          // keep window open when no file-loaded
	_ = m.SetOptionString("profile", "low-latency")        // low latency for RTSP streams
	_ = m.SetOptionString("cache", "no")                   // get latest video for RTSP streams

	// Custom options
	if hwdec != "" {
		_ = m.SetOptionString("hwdec", hwdec)
	}

	_ = m.RequestLogMessages("info")
	_ = m.ObserveProperty(0, PropertyDemuxerCacheTime, mpv.FormatDouble)
	_ = m.ObserveProperty(0, PropertyTimeRemaining, mpv.FormatDouble)

	if err := m.Initialize(); err != nil {
		return Player{}, err
	}

	p := Player{
		ID:       id,
		commandC: make(chan any),
		doneC:    make(chan struct{}),
		closeC:   make(chan struct{}),
	}

	go p.run(ctx, m)

	return p, nil
}

type Player struct {
	ID       string
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
	slog := slog.With("player-id", p.ID)

	defer close(p.doneC)
	defer m.TerminateDestroy()

	eventTicker := time.NewTicker(time.Second)
	defer eventTicker.Stop()

	watchTicker := time.NewTicker(time.Second)
	defer watchTicker.Stop()

	playing := NewState(false)
	file := NewState("")
	volume := NewState(0)
	speed := NewState(1.0)
	watchDog := NewWatchDog(5 * time.Second)

	syncPlayback := func() {
		if playing.V && file.V != "" {
			if err := m.Command([]string{"loadfile", file.V}); err != nil {
				slog.Error("Failed to play file", "error", err)
			}
		} else {
			if err := m.Command([]string{"stop"}); err != nil {
				slog.Error("Failed to stop", "error", err)
			}
		}
		watchDog.Ping()
	}
	playing.AddEffect(syncPlayback)
	file.AddEffect(syncPlayback)

	volume.AddEffect(func() {
		if err := m.SetProperty("volume", mpv.FormatInt64, int64(volume.V)); err != nil {
			slog.Error("Failed to set volume", "error", err)
		}
	})

	speed.AddEffect(func() {
		if err := m.SetProperty("speed", mpv.FormatDouble, speed.V); err != nil {
			slog.Error("Failed to set speed", "error", err)
		}
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.closeC:
			return
		case <-watchTicker.C:
			if watchDog.Dead() {
				syncPlayback()
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
				case mpv.EventNone, mpv.EventShutdown:
					break eventLoop
				case mpv.EventPropertyChange:
					prop := e.Property()
					slog.Debug("property-change", "name", prop.Name, "data", prop.Data)

					switch prop.Name {
					case PropertyDemuxerCacheTime:
						watchDog.Ping()
					case PropertyTimeRemaining:
						if data, ok := prop.Data.(float64); ok {
							if data > MaxPlaybackDelay {
								speed.Update(1.5)
							} else {
								speed.Update(1)
							}
						}
					}
				case mpv.EventLogMsg:
					msg := e.LogMessage()
					switch msg.Level {
					case "fatal", "error":
						slog.Error(msg.Text, "prefix", msg.Prefix)
					case "warn":
						slog.Warn(msg.Text, "prefix", msg.Prefix)
					case "info":
						slog.Info(msg.Text, "prefix", msg.Prefix)
					}
				default:
					slog.Debug("MPV event", "event-id", e.EventID)
				}
			}
		case c := <-p.commandC:
			switch c := c.(type) {
			case CommandPlayback:
				playing.Update(c.Playing)
			case CommandLoad:
				file.Update(c.File)
			case CommandVolume:
				volume.Update(c.Volume)
			}
		}
	}
}

func NewWatchDog(timeout time.Duration) *WatchDog {
	return &WatchDog{
		lastPing: time.Now(),
		timeout:  timeout,
	}
}

type WatchDog struct {
	lastPing time.Time
	timeout  time.Duration
}

func (w *WatchDog) Ping() {
	w.lastPing = time.Now()
}

func (w *WatchDog) Dead() bool {
	return w.lastPing.Add(w.timeout).Before(time.Now())
}
