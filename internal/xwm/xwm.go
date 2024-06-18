package xwm

import (
	"context"
	"log/slog"

	"github.com/jezek/xgb"
	"github.com/thejerf/suture/v4"
)

type Msg any

type Model interface {
	Init(*xgb.Conn) (Model, Cmd)
	Update(*xgb.Conn, Msg) (Model, Cmd)
	Render(*xgb.Conn) error
}

type Cmd func() Msg

func Quit() Msg {
	return QuitMsg{}
}

type QuitMsg struct{}

func NewProgram(initialModel Model) Program {
	return Program{
		initialModel: initialModel,
		msgC:         make(chan Msg),
	}
}

type Program struct {
	initialModel Model
	msgC         chan Msg
}

func (p Program) String() string {
	return "xwm.Program"
}

func (p Program) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := xgb.NewConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	cmdC := make(chan Cmd)

	p.listenEvents(ctx, conn)
	p.handleCommands(ctx, cmdC)

	model, cmd := p.initialModel.Init(conn)
	if cmd != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case cmdC <- cmd:
		}
	}

	for {
		// Message
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-p.msgC:
			// Handle
			switch msg.(type) {
			case QuitMsg:
				return suture.ErrTerminateSupervisorTree
			}

			// Update
			model, cmd = model.Update(conn, msg)
			if cmd != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case cmdC <- cmd:
				}
			}

			// Render
			if err := model.Render(conn); err != nil {
				slog.Error("Failed to render", "error", err)
			}
		}
	}
}

func (p Program) Send(ctx context.Context, msg Msg) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.msgC <- msg:
		return nil
	}
}

func (p Program) Quit(ctx context.Context) error {
	return p.Send(ctx, Quit())
}

func (p Program) listenEvents(ctx context.Context, conn *xgb.Conn) {
	go func() {
		for {
			ev, err := conn.WaitForEvent()
			if ev == nil && err == nil {
				slog.Debug("exit: no event or error")
				return
			}

			if err != nil {
				slog.Error("failed to read event", "error", err)
				return
			}

			p.Send(ctx, ev)
		}
	}()
}

func (p Program) handleCommands(ctx context.Context, cmds chan Cmd) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case cmd := <-cmds:
				if cmd == nil {
					continue
				}

				go func() {
					msg := cmd()
					p.Send(ctx, msg)
				}()
			}
		}
	}()
}
