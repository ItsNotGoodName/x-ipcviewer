package xwm

import (
	"fmt"
	"log"

	vlc "github.com/adrg/libvlc-go/v3"
	"github.com/jezek/xgb/xproto"
)

func init() {
	// Initialize libVLC module.
	err := vlc.Init("--quiet")
	if err != nil {
		log.Fatalln(err)
	}
}

// Window is concurrent safe vlc wrapper.
type Window struct {
	KeyPressEventC chan<- xproto.KeyPressEvent
	WID            xproto.Window
	playerFnC      chan<- func(*vlc.Player)
}

func (w *Window) AccessPlayer(fn func(player *vlc.Player) error) error {
	err := make(chan error)
	w.playerFnC <- func(player *vlc.Player) {
		err <- fn(player)
	}
	return <-err
}

func NewWindow(wid xproto.Window) (Window, error) {
	player, err := vlc.NewPlayer()
	if err != nil {
		return Window{}, err
	}

	if err := player.SetXWindow(uint32(wid)); err != nil {
		return Window{}, err
	}

	if err := player.SetMouseInput(false); err != nil {
		return Window{}, err
	}

	if err := player.SetKeyInput(false); err != nil {
		return Window{}, err
	}

	if err := player.SetMute(true); err != nil {
		return Window{}, err
	}

	playerEvm, err := player.EventManager()
	if err != nil {
		return Window{}, nil
	}

	keyPressEventC := make(chan xproto.KeyPressEvent)
	playerFnC := make(chan func(player *vlc.Player))

	go handlePlayer(player, playerEvm, keyPressEventC, playerFnC)

	return Window{
		KeyPressEventC: keyPressEventC,
		WID:            wid,
		playerFnC:      playerFnC,
	}, nil
}

func handlePlayer(player *vlc.Player, playerEvm *vlc.EventManager, keyPressEventC <-chan xproto.KeyPressEvent, playerFnC chan func(player *vlc.Player)) {
	for {
		select {
		case ev := <-keyPressEventC:
			fmt.Println("window child:", ev.Child)
			player.Play()
		case fn := <-playerFnC:
			fn(player)
		}
	}
}
