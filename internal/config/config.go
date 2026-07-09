// Package config загружает конфигурацию сервиса из yaml-файла
// с возможностью переопределения значений переменными окружения.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Env — окружение, в котором запущен сервис.
const (
	EnvLocal = "local"
	EnvProd  = "prod"
)

// Config — корневая конфигурация сервиса.
type Config struct {
	Env  string     `yaml:"env" env:"APP_ENV" env-default:"local"`
	HTTP HTTPConfig `yaml:"http"`
}

// HTTPConfig — настройки HTTP-сервера.
type HTTPConfig struct {
	Port            int           `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

// Load читает конфигурацию из файла path и окружения.
// Если файл не существует, используются только переменные окружения и значения по умолчанию.
func Load(path string) (*Config, error) {
	var cfg Config

	if _, err := os.Stat(path); err == nil {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("read config %q: %w", path, err)
		}
		return &cfg, nil
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read config from env: %w", err)
	}

	return &cfg, nil
}
