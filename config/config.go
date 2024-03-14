package config

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/ItsNotGoodName/x-ipcviewer/mosaic"
	"github.com/ItsNotGoodName/x-ipcviewer/mpv"
	"github.com/spf13/viper"
)

type Config struct {
	Background          bool
	ConfigWatchExit     bool
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
	Name       string
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
		if cfg.Windows[i].Name == "" {
			cfg.Windows[i].Name = strconv.Itoa(i)

			name, err := parseHostname(cfg.Windows[i].Main)
			if err != nil {
				continue
			}

			cfg.Windows[i].Name = name
		}
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

func parseHostname(maybeUrl string) (string, error) {
	u, err := url.Parse(maybeUrl)
	if err != nil {
		return "", err
	}

	return u.Hostname(), nil
}
