package speech

import (
	"fmt"
	"strings"

	"diplomaMeetHelper/internal/adapters/speech/mock"
	"diplomaMeetHelper/internal/config"
	"diplomaMeetHelper/internal/usecase"
)

func NewClient(cfg config.SpeechConfig) (usecase.SpeechClient, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "mock":
		return mock.NewSpeechClient(cfg.Mock.Delay, cfg.Mock.ErrorRate), nil
	default:
		return nil, fmt.Errorf("unsupported speech provider: %q (supported: mock)", cfg.Provider)
	}
}
