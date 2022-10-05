package mpv

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/ItsNotGoodName/mpvipc"
	"github.com/ItsNotGoodName/x-ipc-viewer/xwm"
	"github.com/google/uuid"
	"github.com/jezek/xgb/xproto"
)

const (
	event_demuxer_cache_idle uint = iota + 1
	event_demuxer_cache_time
)

type Player struct {
	cmd        *exec.Cmd
	conn       *mpvipc.Connection
	streamC    chan string
	socketPath string
}

func NewPlayer(wid xproto.Window) (xwm.Player, error) {
	socketPath := fmt.Sprintf("/tmp/x-ipc-viewer-mpv-%s", uuid.New())

	// Start mpv
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
		"--loop-file=inf",
		fmt.Sprintf("--input-unix-socket=%s", socketPath),
	)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Open connection
	var conn *mpvipc.Connection
	eventC := make(chan *mpvipc.Event, 50)
	for {
		conn = mpvipc.NewConnection(socketPath)
		if err := conn.Open(eventC); err == nil {
			fmt.Println("connected to " + socketPath)
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Setup event observers
	_, err := conn.Call("observe_property", event_demuxer_cache_idle, "demuxer-cache-idle")
	if err != nil {
		return nil, err
	}
	_, err = conn.Call("observe_property", event_demuxer_cache_time, "demuxer-cache-time")
	if err != nil {
		return nil, err
	}

	p := Player{
		cmd:        cmd,
		conn:       conn,
		streamC:    make(chan string),
		socketPath: socketPath,
	}

	go watch(p, eventC)

	return p, nil
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
	if err == nil {
		p.streamC <- stream
	}
	return err
}

func (p Player) Stop() error {
	_, err := p.conn.Call("stop")
	if err == nil {
		p.streamC <- ""
	}
	return err
}

func (p Player) Release() {
	if p.conn.IsClosed() {
		return
	}

	if err := p.conn.Close(); err != nil {
		log.Println("mpv.Player.Release:", err)
	}

	if err := p.cmd.Process.Kill(); err != nil {
		log.Println("mpv.Player.Release:", err)
	}

	if err := os.Remove(p.socketPath); err != nil {
		log.Println("mpv.Player.Release:", err)
	}
}
