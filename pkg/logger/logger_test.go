package logger_test

import (
	"testing"

	"diplomaMeetHelper/pkg/logger"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		format  string
		wantErr bool
	}{
		{
			name:    "json format with info level",
			level:   "info",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "console format with debug level",
			level:   "debug",
			format:  "console",
			wantErr: false,
		},
		{
			name:    "fallback to info on invalid level",
			level:   "non_existing_level",
			format:  "console",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := logger.New(tt.level, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if log == nil {
				t.Errorf("New() returned nil logger")
			}
		})
	}
}
