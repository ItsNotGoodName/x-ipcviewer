package xwm

import "github.com/jezek/xgb/xproto"

// Player handles displaying a stream to a X window.
type Player interface {
	// Mute or unmute audio.
	Mute(mute bool) error
	// Play this stream.
	Play(stream string) error
	// Stop playing this stream.
	Stop() error
	// Release held resources.
	Release()
}

type PlayerFactory func(wid xproto.Window) (Player, error)

// PlayerCache prevents redundant calls to Player.
type PlayerCache struct {
	player Player
	muted  bool
	stream string
}

func NewPlayerCache(player Player) *PlayerCache {
	return &PlayerCache{player: player}
}

func (pc *PlayerCache) Mute(mute bool) error {
	if mute == pc.muted {
		return nil
	}

	if err := pc.player.Mute(mute); err != nil {
		return err
	}

	pc.muted = mute

	return nil
}

func (pc *PlayerCache) Play(stream string) error {
	if stream == pc.stream {
		return nil
	}

	if err := pc.player.Play(stream); err != nil {
		return err
	}

	pc.stream = stream

	return nil
}

func (pc *PlayerCache) Stop() error {
	if pc.stream == "" {
		return nil
	}

	if err := pc.player.Stop(); err != nil {
		return err
	}

	pc.stream = ""

	return nil
}

func (pc *PlayerCache) Release() {
	pc.player.Release()
}
