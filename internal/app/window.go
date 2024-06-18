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
	m := NewMpv(wid)
	_ = m.RequestLogMessages("info")
	_ = m.ObserveProperty(0, "pause", mpv.FormatFlag)

	_ = m.SetPropertyString("input-default-bindings", "yes")
	_ = m.SetOptionString("input-vo-keyboard", "yes")
	_ = m.SetOption("input-cursor", mpv.FormatFlag, true)
	_ = m.SetOption("osc", mpv.FormatFlag, true)
	_ = m.SetOption("force-window", mpv.FormatFlag, true)
	_ = m.SetOption("idle", mpv.FormatFlag, true)
	_ = m.SetOptionString("loop-file", "inf")

	if err := m.Initialize(); err != nil {
		return Window{}, err
	}

	return Window{
		wid:   wid,
		mpv:   m,
		main:  os.Args[1],
		sub:   "",
		flags: []string{},
	}, nil
}

type Window struct {
	wid   xproto.Window
	mpv   *mpv.Mpv
	main  string
	sub   string
	flags []string
}

func (w Window) String() string {
	return fmt.Sprintf("xwm.Window(wid=%d)", w.wid)
}

func (w Window) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Println("i")

	if err := w.mpv.Command([]string{"loadfile", w.main}); err != nil {
		return err
	}

	eventC := w.listenEvents(ctx)

	for {
		select {
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

func (w Window) listenEvents(ctx context.Context) chan *mpv.Event {
	eventC := make(chan *mpv.Event)
	go func() {
		for {
			e := w.mpv.WaitEvent(10000)
			if e.Error != nil {
				slog.Error("Failed to listen for events", "error", e.Error, "window", w.String())
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
