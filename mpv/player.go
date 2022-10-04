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

type Player struct {
	cmd  *exec.Cmd
	conn *mpvipc.Connection
}

func NewPlayer(wid xproto.Window) (xwm.Player, error) {
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

	return &Player{
		cmd:  cmd,
		conn: conn,
	}, nil
}

func (p Player) Mute(mute bool) error {
	var value int
	if mute {
		value = 0
	} else {
		value = 100
	}

	return p.conn.Set("volume", value)
}

func (p Player) Play(stream string) error {
	_, err := p.conn.Call("loadfile", stream)
	return err
}

func (p Player) Stop() error {
	_, err := p.conn.Call("stop")
	return err
}

func (p Player) Release() {
	if err := p.conn.Close(); err != nil {
		log.Println("mpv.Window.Release:", err)
	}

	if err := p.cmd.Process.Kill(); err != nil {
		log.Println("mpv.Window.Release:", err)
	}
}
