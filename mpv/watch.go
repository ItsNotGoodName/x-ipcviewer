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

func watch(p Player, eventC chan *mpvipc.Event) {
	var stream string
	var shouldPlay bool
	var isRtspStream bool

	var isPlaying bool
	reloadStreamC := make(chan struct{}, 1)
	updateC := make(chan struct{}, 1)
	pingD := 5 * time.Second
	pingT := time.NewTicker(pingD)

	for {
		select {
		case <-updateC:
			isRtspStream = strings.HasPrefix(stream, "rtsp://")
			shouldPlay = stream != ""
			if shouldPlay {
				pingT.Reset(pingD)
			} else {
				pingT.Stop()
			}
		default:
		}

		select {
		case stream = <-p.streamC:
			flag(updateC)
		case <-reloadStreamC:
			if shouldPlay {
				log.Printf("mpv.watch: %s: reloading", p.socketPath)
				_, err := p.conn.Call("loadfile", stream)
				if err != nil {
					log.Printf("mpv watch: %s: reloading: %s", p.socketPath, err)
				}
			}
		case <-pingT.C:
			log.Printf("mpv.watch: %s: queuing reload: ping timeout", p.socketPath)
			flag(reloadStreamC)
		case event, ok := <-eventC:
			if !ok {
				return
			}

			switch event.Name {
			case "end-file":
				isPlaying = false
				log.Printf("mpv.watch: %s: stopped", p.socketPath)
			case "file-loaded":
				isPlaying = true
				log.Printf("mpv.watch: %s: playing", p.socketPath)
			default:
				if event.ID == event_demuxer_cache_time || event.Name == "start-file" || event.Name == "idle" {
					// Ping
					pingT.Reset(pingD)
				} else if event.ID == event_demuxer_cache_idle && isRtspStream && isPlaying && event.Data != nil && event.Data.(bool) {
					// Reload stream if cache is idle and is a rtsp stream
					log.Printf("mpv.watch: %s: queuing reload: no longer caching", p.socketPath)
					flag(reloadStreamC)
				}
			}

		}
	}
}
