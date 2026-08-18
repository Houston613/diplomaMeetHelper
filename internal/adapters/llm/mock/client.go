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

	return fmt.Sprintf("Краткое содержание встречи: согласованы ключевые решения по архитектуре. Длина исходного текста: %d символов.", len(transcript)), nil
}

func (c *LLMClient) AskQuestion(ctx context.Context, contextText string, question string) (string, error) {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return fmt.Sprintf("Ответ на вопрос %q: на основе материалов встречи подтверждаем принятие всех ключевых решений.", question), nil
}
