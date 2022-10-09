package xwm

import (
	"log"

	"github.com/jezek/xgb/xproto"
)

type Window struct {
	wid        xproto.Window
	player     Player
	mainStream string
	subStream  string
	background bool
}

func NewWindow(wid xproto.Window, player Player, mainStream, subStream string, background bool) Window {
	if subStream == "" {
		subStream = mainStream
	}

	return Window{
		wid:        wid,
		player:     player,
		mainStream: mainStream,
		subStream:  subStream,
		background: background,
	}
}

func (c Window) Show(focus, fullscreen bool) {
	var stream string
	if fullscreen {
		stream = c.mainStream
	} else {
		stream = c.subStream
	}
	if err := c.player.Play(stream); err != nil {
		log.Println("xwm.Window.Show: Play:", err)
	}

	if err := c.player.Mute(!focus); err != nil {
		log.Println("xwm.Window.Show: Mute:", err)
	}
}

func (c Window) Hide() {
	if c.background {
		if err := c.player.Play(c.subStream); err != nil {
			log.Println("xwm.Window.Show: Play:", err)
		}

		if err := c.player.Mute(true); err != nil {
			log.Println("xwm.Window.Show: Mute:", err)
		}

		return
	}

	if err := c.player.Stop(); err != nil {
		log.Println("xwm.Window.Show: Stop:", err)
	}
}

func (c Window) Release() {
	c.player.Release()
}
