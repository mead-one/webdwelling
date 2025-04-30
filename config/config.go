package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port int `yaml:"port"`
	BasePath string `yaml:"base_path"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	err = yaml.NewDecoder(file).Decode(&cfg)
	return &cfg, err
}
