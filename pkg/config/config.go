package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Port     int         `yaml:"port" env-default:"8080"`
	SavePath string      `yaml:"savePath" env-default:"uploads"`
	HTTP     *HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	MaxStreamBuffer int   `yaml:"maxStreamBuffer"`
	MaxUploadSize   int64 `yaml:"maxUploadSize"`
	DefaultPage     int   `yaml:"defaultPage"`
	DefaultSize     int   `yaml:"defaultSize"`
}

func MustLoad(configPath string) *Config {
	var conf Config

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic("failed to read config: " + err.Error())
	}

	if err = yaml.Unmarshal(data, &conf); err != nil {
		panic("failed to unmarshal config: " + err.Error())
	}

	return &conf
}
