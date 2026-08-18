package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App    AppConfig    `mapstructure:"app"`
	DB     DBConfig     `mapstructure:"db"`
	Speech SpeechConfig `mapstructure:"speech"`
	LLM    LLMConfig    `mapstructure:"llm"`
	Log    LogConfig    `mapstructure:"log"`
}

type AppConfig struct {
	Workers    int           `mapstructure:"workers"`
	JobTimeout time.Duration `mapstructure:"job_timeout"`
}

type DBConfig struct {
	DSN string `mapstructure:"dsn"`
}

type SpeechConfig struct {
	Provider string     `mapstructure:"provider"`
	Mock     MockConfig `mapstructure:"mock"`
}

type LLMConfig struct {
	Provider string     `mapstructure:"provider"`
	Mock     MockConfig `mapstructure:"mock"`
}

type MockConfig struct {
	Delay     time.Duration `mapstructure:"delay"`
	ErrorRate int           `mapstructure:"error_rate"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.DB.DSN) == "" {
		return errors.New("config: db.dsn cannot be empty")
	}
	if c.App.Workers <= 0 {
		return errors.New("config: app.workers must be greater than 0")
	}
	if c.App.JobTimeout <= 0 {
		return errors.New("config: app.job_timeout must be positive")
	}
	return nil
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("app.workers", 3)
	v.SetDefault("app.job_timeout", "120s")
	v.SetDefault("db.dsn", "postgres://postgres:postgrespassword@localhost:5432/meethelper?sslmode=disable&search_path=meethelper")
	v.SetDefault("speech.provider", "mock")
	v.SetDefault("speech.mock.delay", "500ms")
	v.SetDefault("speech.mock.error_rate", 0)
	v.SetDefault("llm.provider", "mock")
	v.SetDefault("llm.mock.delay", "500ms")
	v.SetDefault("llm.mock.error_rate", 0)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")

	v.SetConfigFile(configPath)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// Non-critical if config file does not exist, defaults will be used
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
