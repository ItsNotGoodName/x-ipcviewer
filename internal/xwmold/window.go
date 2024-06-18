package xwmold

import (
	"context"
	"fmt"
	"os"

	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb/xproto"
)

type Window struct {
	wid   xproto.Window
	main  string
	sub   string
	flags []string
}

func (w Window) String() string {
	return fmt.Sprintf("xwm.Window(wid=%d)", w.wid)
}

func (w Window) Serve(ctx context.Context) error {
	m := NewMpv(w.wid)
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
		return err
	}

	if err := m.Command([]string{"loadfile", os.Args[1]}); err != nil {
		return err
	}

	for {
		e := m.WaitEvent(10000)

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

		if e.Error != nil {
			fmt.Println("error:", e.Error)
		}
	}
}
