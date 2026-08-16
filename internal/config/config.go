package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DB  DBConfig  `mapstructure:"db"`
	Log LogConfig `mapstructure:"log"`
}

type DBConfig struct {
	DSN string `mapstructure:"dsn"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.DB.DSN) == "" {
		return errors.New("config: db.dsn cannot be empty")
	}
	return nil
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("db.dsn", "postgres://postgres:postgrespassword@localhost:5432/meethelper?sslmode=disable&search_path=meethelper")
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
