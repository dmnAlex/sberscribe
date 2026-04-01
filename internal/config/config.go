package config

import (
	"encoding/json"
	"flag"
	"os"

	"dario.cat/mergo"
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

type Config struct {
	LogLevel       string `env:"LOG_LEVEL" json:"log_level"`
	DatabaseDSN    string `env:"DATABASE_DSN" json:"database_dsn"`
	MigrationsPath string `env:"MIGRATIONS_PATH" json:"migrations_path"`

	TelegramToken string `env:"TELEGRAM_TOKEN,required" json:"-"`

	SaluteSpeechClientSecret string `env:"SALUTE_SPEECH_CLIENT_SECRET,required" json:"-"`

	GigaChatClientSecret string `env:"GIGA_CHAT_CLIENT_SECRET,required" json:"-"`
	GigaChatModel        string `env:"GIGACHAT_MODEL" json:"giga_chat_model"`

	CACertPath string `env:"CA_CERT_PATH,required" json:"-"`
}

func New() (*Config, error) {
	cfg := &Config{}
	var configPath string
	flag.StringVar(&configPath, "c", "", "path to config file")
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database dsn")
	flag.StringVar(&cfg.MigrationsPath, "m", "", "migrations path")
	flag.Parse()

	if envPath := os.Getenv("CONFIG"); envPath != "" {
		configPath = envPath
	}

	if configPath != "" {
		jsonCfg, err := loadJSON(configPath)
		if err != nil {
			return nil, errors.Wrap(err, "load json")
		}

		if err := mergo.Merge(cfg, jsonCfg); err != nil {
			return nil, errors.Wrap(err, "merge json configs")
		}
	}

	if err := godotenv.Load(); err != nil {
		return nil, errors.Wrap(err, "load env file")
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	setDefaults(cfg)

	return cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "./migrations"
	}
	if cfg.GigaChatModel == "" {
		cfg.GigaChatModel = "GigaChat"
	}
}

func loadJSON(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "open file")
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, errors.Wrap(err, "decode json")
	}

	return &cfg, nil
}
