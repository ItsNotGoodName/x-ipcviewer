package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb/xproto"
)

func NewWindow(wid xproto.Window) (Window, error) {
	m := mpv.New()

	_ = m.SetOption("wid", mpv.FormatInt64, int64(wid))    // bind to x window
	_ = m.SetOptionString("input-vo-keyboard", "no")       // passthrough keyboard input to sub x window
	_ = m.SetOption("input-cursor", mpv.FormatFlag, false) // passthrough mouse input to sub x window
	_ = m.SetOption("osc", mpv.FormatFlag, false)          // don't render on screen ui
	_ = m.SetOption("force-window", mpv.FormatFlag, true)  // render empty video when no file-loaded
	_ = m.SetOption("idle", mpv.FormatFlag, true)          // keep window open when no file-loaded
	_ = m.SetOptionString("loop-file", "inf")              // loop video

	_ = m.RequestLogMessages("info")
	// _ = m.ObserveProperty(0, "pause", mpv.FormatFlag)

	_ = m.SetOptionString("input-vo-keyboard", "no")

	if err := m.Initialize(); err != nil {
		return Window{}, err
	}

	return Window{
		mpv:    m,
		main:   os.Args[1],
		sub:    "",
		flags:  []string{},
		eventC: make(chan any),
	}, nil
}

type Window struct {
	mpv    *mpv.Mpv
	main   string
	sub    string
	flags  []string
	eventC chan any
}

func (w Window) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := w.mpv.Command([]string{"loadfile", w.main}); err != nil {
		return err
	}

	eventC := w.listenEvents(ctx)

	for {
		select {
		case e := <-w.eventC:
			switch e.(type) {
			}
		case e := <-eventC:
			switch e.EventID {
			case mpv.EventPropertyChange:
				prop := e.Property()
				value := prop.Data.(int)
				fmt.Println("property:", prop.Name, value)
			case mpv.EventFileLoaded:
				p, err := w.mpv.GetProperty("media-title", mpv.FormatString)
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
					return nil
				} else if ef.Reason == mpv.EndFileError {
					fmt.Println("error:", ef.Error)
				}
			case mpv.EventShutdown:
				fmt.Println("shutdown:", e.EventID)
				return nil
			default:
				fmt.Println("event:", e.EventID)
			}
		}
	}
}

func (w Window) Send(ctx context.Context, cmd any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case w.eventC <- cmd:
		return nil
	}
}

func (w Window) listenEvents(ctx context.Context) chan *mpv.Event {
	eventC := make(chan *mpv.Event)
	go func() {
		for {
			e := w.mpv.WaitEvent(10000)
			if e.Error != nil {
				slog.Error("Failed to listen for events", "error", e.Error)
				return
			}

			select {
			case <-ctx.Done():
				return
			case eventC <- e:
			}
		}
	}()
	return eventC
}
