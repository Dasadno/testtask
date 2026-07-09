// Package config загружает конфигурацию сервиса из yaml-файла
// с возможностью переопределения значений переменными окружения.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Env — окружение, в котором запущен сервис.
const (
	EnvLocal = "local"
	EnvProd  = "prod"
)

// Config — корневая конфигурация сервиса.
type Config struct {
	Env      string         `yaml:"env" env:"APP_ENV" env-default:"local"`
	HTTP     HTTPConfig     `yaml:"http"`
	Postgres PostgresConfig `yaml:"postgres"`
}

// HTTPConfig — настройки HTTP-сервера.
type HTTPConfig struct {
	Port            int           `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

// PostgresConfig — настройки подключения к PostgreSQL.
// Пароль передаётся только через переменную окружения, в yaml его не храним.
type PostgresConfig struct {
	Host           string `yaml:"host" env:"POSTGRES_HOST" env-default:"localhost"`
	Port           int    `yaml:"port" env:"POSTGRES_PORT" env-default:"5432"`
	User           string `yaml:"user" env:"POSTGRES_USER" env-default:"postgres"`
	Password       string `env:"POSTGRES_PASSWORD" env-default:"postgres"`
	Database       string `yaml:"database" env:"POSTGRES_DB" env-default:"subscriptions"`
	SSLMode        string `yaml:"ssl_mode" env:"POSTGRES_SSLMODE" env-default:"disable"`
	MigrationsPath string `yaml:"migrations_path" env:"MIGRATIONS_PATH" env-default:"env/migrations"`
}

// DSN собирает строку подключения к PostgreSQL.
func (c PostgresConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:     c.Database,
		RawQuery: "sslmode=" + c.SSLMode,
	}
	return u.String()
}

// Load читает конфигурацию из файла path и окружения.
// Если файл не существует, используются только переменные окружения и значения по умолчанию.
// Перед чтением подхватывается .env из рабочей директории (если есть),
// уже установленные переменные окружения он не переопределяет.
func Load(path string) (*Config, error) {
	_ = godotenv.Load()

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
