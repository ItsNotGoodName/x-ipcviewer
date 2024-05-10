package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ItsNotGoodName/x-ipcviewer/xcursor"
	"github.com/gen2brain/go-mpv"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func main() {
	x, err := xgb.NewConn()
	if err != nil {
		panic(err)
	}
	defer x.Close()

	screen := xproto.Setup(x).DefaultScreen(x)

	cursor, err := xcursor.CreateCursor(x, xcursor.LeftPtr)
	if err != nil {
		panic(err)
	}

	wid, err := xproto.NewWindowId(x)
	if err != nil {
		panic(err)
	}

	// Create root X window with black background and listen for resize, key presses, and button presses events
	if err := xproto.CreateWindowChecked(x, screen.RootDepth,
		wid, screen.Root,
		0, 0, screen.WidthInPixels/2, screen.HeightInPixels/2, 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask|xproto.CwCursor, // 1, 2, 3
		[]uint32{
			0, // 1
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskButtonPress, // 2
			uint32(cursor), // 3
		}).Check(); err != nil {
		panic(err)
	}

	swid, err := CreateXSubWindow(x, wid)
	if err != nil {
		panic(err)
	}

	m := mpv.New()
	defer m.TerminateDestroy()

	_ = m.RequestLogMessages("info")
	_ = m.ObserveProperty(0, "pause", mpv.FormatFlag)

	err = m.SetPropertyString("input-default-bindings", "yes")
	err = m.SetOptionString("input-vo-keyboard", "yes")
	err = m.SetOption("input-cursor", mpv.FormatFlag, true)
	err = m.SetOption("osc", mpv.FormatFlag, true)
	err = m.SetOption("force-window", mpv.FormatFlag, true)
	err = m.SetOption("idle", mpv.FormatFlag, true)
	err = m.SetOptionString("loop-file", "inf")
	err = m.SetOptionString("wid", strconv.Itoa(int(swid)))

	if err := m.Initialize(); err != nil {
		panic(err)
	}

	err = m.Command([]string{"loadfile", os.Args[1]})
	if err != nil {
		panic(err)
	}
loop:
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
				break loop
			} else if ef.Reason == mpv.EndFileError {
				fmt.Println("error:", ef.Error)
			}
		case mpv.EventShutdown:
			fmt.Println("shutdown:", e.EventID)
			break loop
		case mpv.EventPlaybackRestart:
			// Show root X window
			if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
				// xproto.DestroyWindow(x, wid)
				panic(err)
			}
		default:
			fmt.Println("event:", e.EventID)
		}

		if e.Error != nil {
			fmt.Println("error:", e.Error)
		}
	}
}

func CreateXSubWindow(x *xgb.Conn, root xproto.Window) (xproto.Window, error) {
	// Generate X window id
	wid, err := xproto.NewWindowId(x)
	if err != nil {
		return 0, err
	}

	// Create X window in root
	if err := xproto.CreateWindowChecked(x, xproto.WindowClassCopyFromParent,
		wid, root,
		0, 0, 256, 256, 0,
		xproto.WindowClassInputOutput, xproto.WindowClassCopyFromParent, 0, []uint32{}).Check(); err != nil {
		return 0, err
	}

	// Show X window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		xproto.DestroyWindow(x, wid)
		return 0, err
	}

	return wid, nil
}
