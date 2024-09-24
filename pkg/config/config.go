package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Db       string         `yaml:"db"`
	Port     int            `yaml:"port" env-default:"8080"`
	Mongo    MongoConfig    `yaml:"mongo"`
	Postgres PostgresConfig `yaml:"postgres"`
}

type PostgresConfig struct {
	Host     string   `yaml:"host" env-default:"localhost"`
	Port     int      `yaml:"port" env-default:"5432"`
	User     string   `yaml:"user" env-default:"postgres"`
	Password string   `yaml:"password" env-default:"postgres"`
	Database string   `yaml:"database" env-default:"db"`
	Field    string   `yaml:"field" env-default:"src"`
	Tables   []string `yaml:"tables" env-default:"table_name"`
}

type MongoConfig struct {
	URI         string   `yaml:"URI"`
	Name        string   `yaml:"name"`
	Collections []string `yaml:"collections"`
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
