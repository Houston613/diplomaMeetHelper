package llm

import (
	"fmt"
	"strings"

	"diplomaMeetHelper/internal/adapters/llm/mock"
	"diplomaMeetHelper/internal/config"
	"diplomaMeetHelper/internal/usecase"
)

type Client interface {
	usecase.LLMClient
	usecase.ChatLLMClient
}

func NewClient(cfg config.LLMConfig) (Client, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "mock":
		return mock.NewLLMClient(cfg.Mock.Delay, cfg.Mock.ErrorRate), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %q (supported: mock)", cfg.Provider)
	}
}
