package config

import (
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Default location of the config file, relative to the working directory
const DefaultPath = "config.yaml"

// Environment variable holding the secret used to sign JWT tokens
const JWTSecretEnv = "JWT_SECRET"

// Minimum length of the JWT secret in bytes
const minJWTSecretLength = 32

type Config struct {
	Port          int    `yaml:"port"`
	BasePath      string `yaml:"base_path"`
	SecureCookies bool   `yaml:"secure_cookies"`
	JWTSecret     []byte `yaml:"-"` // Not in yaml file, loaded from environment
}

// Defaults for any option that is commented out or missing from the config file
func defaultConfig() Config {
	return Config{
		Port:          1323,
		BasePath:      "",
		SecureCookies: true,
	}
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := defaultConfig()
	// A file containing only comments decodes as io.EOF, which just means "use the defaults"
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	secret := os.Getenv(JWTSecretEnv)
	if len(secret) < minJWTSecretLength {
		return nil, fmt.Errorf("%s must be set to at least %d characters (generate one with `openssl rand -hex 32`)", JWTSecretEnv, minJWTSecretLength)
	}
	cfg.JWTSecret = []byte(secret)

	return &cfg, nil
}
