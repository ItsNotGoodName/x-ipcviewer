package config

import (
	"fmt"

	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/spf13/viper"
)

type Config struct {
	Background          bool
	Layout              Layout
	LayoutManual        []LayoutManual
	LayoutManualWindows []mosaic.LayoutManualWindow `mapstructure:"-"`
	Windows             []Window
}

type Window struct {
	Main string
	Sub  string
}

func Decode(cfg *Config) error {
	if err := viper.Unmarshal(cfg); err != nil {
		return err
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
