package mpv

import (
	"log"
	"strings"
	"time"

	"github.com/ItsNotGoodName/mpvipc"
)

func flag(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

func watch(p Player, eventC <-chan *mpvipc.Event) {
	// modifiable by p.streamC
	var stream string
	var shouldPlay bool
	var isRtspStream bool

	// modifiable by eventC
	var isPlaying bool

	// modifiable by eventC and pingT.C
	pingD := 5 * time.Second
	pingT := time.NewTicker(pingD)

	reloadStreamC := make(chan struct{}, 1)

	for {
		select {
		case <-reloadStreamC:
			if shouldPlay {
				log.Printf("mpv.watch: %s: reloading", p.socketPath)
				_, err := p.conn.Call("loadfile", stream)
				if err != nil {
					log.Printf("mpv.watch: %s: reloading: %s", p.socketPath, err)
				}
			} else {
				log.Printf("mpv.watch: %s: stopping", p.socketPath)
				_, err := p.conn.Call("stop")
				if err != nil {
					log.Printf("mpv.watch: %s: stopping: %s", p.socketPath, err)
				}
			}
		case stream = <-p.streamC:
			isRtspStream = strings.HasPrefix(stream, "rtsp://")
			shouldPlay = stream != ""
			flag(reloadStreamC)
		case <-pingT.C:
			log.Printf("mpv.watch: %s: queuing reload: ping timeout", p.socketPath)
			flag(reloadStreamC)
		case event, ok := <-eventC:
			if !ok {
				pingT.Stop()
				return
			}

			switch event.Name {
			case "start-file":
				log.Printf("mpv.watch: %s: event: %s", p.socketPath, event.Name)
				isPlaying = false
				if isRtspStream {
					pingT.Reset(pingD)
				} else {
					pingT.Stop()
				}
			case "file-loaded":
				log.Printf("mpv.watch: %s: event: %s", p.socketPath, event.Name)
				isPlaying = true
				if isRtspStream {
					pingT.Reset(pingD)
				} else {
					pingT.Stop()
				}
			case "end-file":
				log.Printf("mpv.watch: %s: event: %s", p.socketPath, event.Name)
				isPlaying = false
				pingT.Reset(pingD)
			case "idle":
				log.Printf("mpv.watch: %s: event: %s", p.socketPath, event.Name)
				isPlaying = false
				pingT.Reset(pingD)
			default:
				if isRtspStream {
					if event.ID == event_demuxer_cache_time {
						// Ping
						pingT.Reset(pingD)
					} else if event.ID == event_demuxer_cache_idle && isPlaying && p.lowLatency && event.Data != nil && event.Data.(bool) {
						// Reload stream if cache is idle and is a rtsp stream
						log.Printf("mpv.watch: %s: queuing reload: no longer caching", p.socketPath)
						flag(reloadStreamC)
					}
				}
			}
		}
	}
}
