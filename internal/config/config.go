package config

import (
	"errors"
	"os"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/core"
	"gopkg.in/yaml.v3"
)

func read(filePath string) (Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultConfig, nil
		}
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	err = yaml.NewDecoder(file).Decode(&cfg)
	return cfg, nil
}

func write(filePath string, cfg Config) error {
	filePathTmp := filePath + ".tmp"
	file, err := os.OpenFile(filePathTmp, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	if err := yaml.NewEncoder(file).Encode(cfg); err != nil {
		file.Close()
		return err
	}
	file.Close()

	return os.Rename(filePathTmp, filePath)
}

func NewProvider(filePath string) (Provider, error) {
	if exist, err := core.FileExists(filePath); err != nil {
		return Provider{}, err
	} else if !exist {
		if err := write(filePath, defaultConfig); err != nil {
			return Provider{}, err
		}
	}
	return Provider{
		filePath: filePath,
	}, nil
}

type Provider struct {
	filePath string
}

func (p Provider) GetConfig() (Config, error) {
	return read(p.filePath)
}

func (p Provider) UpdateConfig(fn func(cfg Config) (Config, error)) error {
	cfg, err := read(p.filePath)
	if err != nil {
		return err
	}

	cfg, err = fn(cfg)
	if err != nil {
		return err
	}

	return write(p.filePath, cfg)
}
