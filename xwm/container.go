package xwm

import "github.com/jezek/xgb/xproto"

type ContainerConfig struct {
	MainStream string
	SubStream  string
}

type Container struct {
	WID        xproto.Window
	Window     Window
	MainStream string
	SubStream  string
}

func (c *Container) DefaultStream() string {
	if c.SubStream != "" {
		return c.SubStream
	}
	return c.MainStream
}
