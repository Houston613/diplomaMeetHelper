package config_test

import (
	"os"
	"testing"
	"time"

	"diplomaMeetHelper/internal/config"
)

func TestConfig_DefaultValues(t *testing.T) {
	cfg, err := config.Load("non_existent_config.yaml")
	if err != nil {
		t.Fatalf("expected no error loading defaults, got: %v", err)
	}

	if cfg.App.Workers != 3 {
		t.Errorf("expected 3 workers by default, got %d", cfg.App.Workers)
	}
	if cfg.App.JobTimeout != 120*time.Second {
		t.Errorf("expected 120s timeout by default, got %v", cfg.App.JobTimeout)
	}
	if cfg.DB.DSN == "" {
		t.Errorf("expected non-empty default DSN")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected info log level by default, got %s", cfg.Log.Level)
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			mutate:  func(c *config.Config) {},
			wantErr: false,
		},
		{
			name: "empty DSN",
			mutate: func(c *config.Config) {
				c.DB.DSN = "  "
			},
			wantErr: true,
		},
		{
			name: "zero workers",
			mutate: func(c *config.Config) {
				c.App.Workers = 0
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			mutate: func(c *config.Config) {
				c.App.JobTimeout = -10 * time.Second
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load("non_existent_config.yaml")
			if err != nil {
				t.Fatalf("failed to load base config: %v", err)
			}
			tt.mutate(cfg)

			err = cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_EnvOverride(t *testing.T) {
	_ = os.Setenv("APP_WORKERS", "8")
	_ = os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		_ = os.Unsetenv("APP_WORKERS")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := config.Load("non_existent_config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Workers != 8 {
		t.Errorf("expected 8 workers from env APP_WORKERS, got %d", cfg.App.Workers)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected debug log level from env LOG_LEVEL, got %s", cfg.Log.Level)
	}
}
