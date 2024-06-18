package xwm

import (
	"fmt"

	"github.com/jezek/xgb"
)

var errQuit = fmt.Errorf("quit")

func Quit(conn *xgb.Conn) error {
	return errQuit
}

// Msg contain data from the result of a IO operation. Msgs trigger the update
// function and, henceforth, the UI.
type Msg interface{}

type Model interface {
	// Init is the first function that will be called.
	Init()

	// Update is called when a message is received. Use it to inspect messages
	// and, in response, update the model and/or render.
	Update(Msg) (Model, Render)
}

type Render func(conn *xgb.Conn) error
