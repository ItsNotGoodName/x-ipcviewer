package config

import (
	"fmt"

	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/ItsNotGoodName/x-ipc-viewer/mpv"
	"github.com/spf13/viper"
)

type Config struct {
	Background          bool
	Player              Player
	Layout              Layout
	LayoutManual        []LayoutManual
	LayoutManualWindows []mosaic.LayoutManualWindow `mapstructure:"-"`
	Windows             []Window
}

type Player struct {
	GPU        string
	Flags      []string
	LowLatency bool
}

type Window struct {
	Main       string
	Sub        string
	Flags      []string
	LowLatency bool
}

func Decode(cfg *Config) error {
	viper.SetDefault("Player.GPU", mpv.DefaultGPU)
	viper.SetDefault("Player.LowLatency", mpv.DefaultLowLatency)

	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	// Decode Windows
	for i, raw := range viper.Get("Windows").([]any) {
		m := raw.(map[string]any)
		if _, ok := m["lowlatency"]; !ok {
			cfg.Windows[i].LowLatency = cfg.Player.LowLatency
		}
		if _, ok := m["flags"]; !ok {
			cfg.Windows[i].Flags = cfg.Player.Flags
		}
	}

	// Decode LayoutManualWindows
	for i, lmr := range cfg.LayoutManual {
		lmw, err := parseLayoutManualWindow(lmr)
		if err != nil {
			return fmt.Errorf("LayoutManual[%d].%w", i, err)
		}

		cfg.LayoutManualWindows = append(cfg.LayoutManualWindows, lmw)
	}

	return nil
}
