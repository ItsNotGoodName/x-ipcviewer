package config

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/ItsNotGoodName/x-ipcviewer/internal/core"
	"gopkg.in/yaml.v3"
)

func NewYAML(filePath string) YAML {
	return YAML{
		filePath: filePath,
	}
}

type YAML struct {
	filePath string
}

// Exists implements Driver.
func (y YAML) Exists() (bool, error) {
	return core.FileExists(y.filePath)
}

func (y YAML) Read() (Config, error) {
	file, err := os.Open(y.filePath)
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

func (y YAML) Write(cfg Config) error {
	filePathTmp := y.filePath + ".tmp"
	file, err := os.OpenFile(filePathTmp, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	if err := yaml.NewEncoder(file).Encode(cfg); err != nil {
		file.Close()
		return err
	}
	file.Close()

	return os.Rename(filePathTmp, y.filePath)
}

func NewJSON(filePath string) JSON {
	return JSON{
		filePath: filePath,
	}
}

type JSON struct {
	filePath string
}

func (j JSON) Read() (Config, error) {
	file, err := os.Open(j.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultConfig, nil
		}
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	err = json.NewDecoder(file).Decode(&cfg)
	return cfg, nil
}

func (j JSON) Write(cfg Config) error {
	filePathTmp := j.filePath + ".tmp"
	file, err := os.OpenFile(filePathTmp, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(file).Encode(cfg); err != nil {
		file.Close()
		return err
	}
	file.Close()

	return os.Rename(filePathTmp, j.filePath)
}

type Memory struct {
	mu sync.RWMutex
}
