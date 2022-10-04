package xwm

import "github.com/jezek/xgb/xproto"

type WindowConfig struct {
	MainStream string
	SubStream  string
	Background bool
}

type Window struct {
	wid        xproto.Window
	player     Player
	mainStream string
	subStream  string
	background bool
}

func NewWindow(wid xproto.Window, player Player, config WindowConfig) Window {
	return Window{
		wid:        wid,
		player:     player,
		mainStream: config.MainStream,
		subStream:  config.SubStream,
		background: config.Background,
	}
}

func (c Window) Show(focus, fullscreen bool) error {
	var stream string
	if fullscreen {
		stream = c.mainStream
	} else {
		stream = c.subStream
	}
	if err := c.player.Play(stream); err != nil {
		return err
	}

	return c.player.Mute(!focus)
}

func (c Window) Hide() error {
	if c.background {
		if err := c.player.Play(c.subStream); err != nil {
			return err
		}

		return c.player.Mute(true)
	} else {
		return c.player.Stop()
	}
}

func (c Window) Release() {
	c.player.Release()
}
