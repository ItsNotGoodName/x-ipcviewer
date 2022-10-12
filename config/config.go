package config

import (
	"fmt"

	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/ItsNotGoodName/x-ipc-viewer/mpv"
	"github.com/spf13/viper"
)

type Config struct {
	Background          bool
	Layout              Layout
	LayoutManualWindows []mosaic.LayoutManualWindow `mapstructure:"-"`
	Player              Player
	Windows             []Window
}

type Layout string

func (c Layout) IsAuto() bool {
	return c == "" || c == "auto"
}

func (c Layout) IsManual() bool {
	return c == "manual"
}

type Player struct {
	GPU   string
	Flags []string
}

type Window struct {
	Main       string
	Sub        string
	LowLatency bool
	Flags      []string
}

func Parse(cfg *Config) error {
	viper.SetDefault("Player.GPU", mpv.DefaultGPU)

	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	// Parse Windows
	for i := range cfg.Windows {
		cfg.Windows[i].Flags = append(cfg.Player.Flags, cfg.Windows[i].Flags...)
	}

	// Parse LayoutManualWindows
	var clm ConfigLayoutManual
	if err := viper.Unmarshal(&clm); err != nil {
		return err
	}
	for i, lm := range clm.LayoutManual {
		lmw, err := parseLayoutManualWindow(lm)
		if err != nil {
			return fmt.Errorf("LayoutManual[%d].%w", i, err)
		}

		cfg.LayoutManualWindows = append(cfg.LayoutManualWindows, lmw)
	}

	return nil
}
