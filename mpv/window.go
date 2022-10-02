package mpv

import (
	"fmt"
	"log"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/DexterLB/mpvipc"
	"github.com/ItsNotGoodName/x-ipc-viewer/xwm"
	"github.com/jezek/xgb/xproto"
)

var rpcID int32 = 0

type Window struct {
	wid  xproto.Window
	cmd  *exec.Cmd
	conn *mpvipc.Connection
}

func NewWindow(wid xproto.Window) (xwm.Window, error) {
	socketPath := fmt.Sprintf("/tmp/mpv_socket_%d", atomic.AddInt32(&rpcID, 1))

	cmd := exec.Command(
		"mpv",
		fmt.Sprintf("--wid=%d", wid),
		"--vo=gpu",
		"--hwdec=auto",
		"--profile=low-latency",
		"--no-cache",
		"--input-vo-keyboard=no",
		"--no-input-cursor",
		"--no-osc",
		"--force-window",
		"--idle",
		"--input-unix-socket="+socketPath,
	)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var conn *mpvipc.Connection
	for {
		conn = mpvipc.NewConnection(socketPath)
		if err := conn.Open(); err != nil {
			fmt.Println("connected to " + socketPath)
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
	for {
		if err := conn.Open(); err == nil {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return &Window{
		wid:  wid,
		cmd:  cmd,
		conn: conn,
	}, nil
}

func (w *Window) KeyPress(ev xproto.KeyPressEvent) {}

func (w *Window) ButtonPress(ev xproto.ButtonPressEvent) {}

func (w *Window) Mute(mute bool) error {
	var value string
	if mute {
		value = "yes"
	} else {
		value = "no"
	}
	return w.conn.Set("ao-mute", value)
}

func (w *Window) Start(url string) error {
	_, err := w.conn.Call("loadfile", url)
	return err
}

func (w *Window) Stop() error {
	_, err := w.conn.Call("stop")
	return err
}

func (w *Window) Release() {
	if err := w.conn.Close(); err != nil {
		log.Println("mpv.Window.Release:", err)
	}

	if err := w.cmd.Process.Kill(); err != nil {
		log.Println("mpv.Window.Release:", err)
	}
}
