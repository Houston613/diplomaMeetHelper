package mock

import (
	"context"
	"fmt"
	"time"
)

type LLMClient struct {
	delay     time.Duration
	errorRate int
}

func NewLLMClient(delay time.Duration, errorRate int) *LLMClient {
	return &LLMClient{
		delay:     delay,
		errorRate: errorRate,
	}
}

func (c *LLMClient) Summarize(ctx context.Context, transcript string) (string, error) {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return fmt.Sprintf("Краткое содержание встречи: Согласовали решения по архитектуре и выбору стека. Длина исходного текста: %d символов.", len(transcript)), nil
}
