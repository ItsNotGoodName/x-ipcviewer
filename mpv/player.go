package mpv

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/ItsNotGoodName/mpvipc"
	"github.com/ItsNotGoodName/x-ipc-viewer/xwm"
	"github.com/avast/retry-go/v3"
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
	lowLatency bool
}

const DefaultGPU string = "auto"

func NewPlayerFactory(flags []string, gpu string, lowLatency bool) xwm.PlayerFactory {
	return func(wid xproto.Window) (xwm.Player, error) {
		socketPath := fmt.Sprintf("/tmp/x-ipc-viewer-mpv-%s", uuid.New())

		args := []string{
			fmt.Sprintf("--input-unix-socket=%s", socketPath), // mpvipc
			fmt.Sprintf("--wid=%d", wid),                      // bind to x window
			"--input-vo-keyboard=no",                          // passthrough keyboard input to sub x window
			"--no-input-cursor",                               // passthrough mouse input to sub x window
			"--no-osc",                                        // don't render on-screen-ui
			"--force-window",                                  // render empty video when no file-loaded
			"--idle",                                          // keep window open when no file-loaded
			"--loop-file=inf",                                 // loop video
		}

		args = append(args, fmt.Sprintf("--hwdec=%s", gpu))

		if lowLatency {
			args = append(args, "--profile=low-latency", "--no-cache")
		}

		args = append(args, flags...)

		// Start mpv
		cmd := exec.Command("mpv", args...)
		if err := cmd.Start(); err != nil {
			return nil, err
		}

		// Open connection
		conn := mpvipc.NewConnection(socketPath)
		var eventC <-chan *mpvipc.Event
		if err := retry.Do(func() error {
			var err error
			eventC, err = conn.Open(50)
			return err
		}, retry.Attempts(50), retry.DelayType(retry.FixedDelay)); err != nil {
			return nil, err
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
			streamC:    make(chan string, 1),
			socketPath: socketPath,
			lowLatency: lowLatency,
		}

		go p.watch(eventC)

		return p, nil
	}
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
	for {
		select {
		case p.streamC <- stream:
			return nil
		case <-p.streamC:
		}
	}
}

func (p Player) Stop() error {
	for {
		select {
		case p.streamC <- "":
			return nil
		case <-p.streamC:
		}
	}
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
