package mock

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
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

	if strings.Contains(transcript, "[SIMULATE_LLM_ERROR]") {
		return "", errors.New("simulated LLM summarization failure")
	}

	if c.errorRate > 0 && rand.Intn(100) < c.errorRate {
		return "", fmt.Errorf("LLM summarization service temporarily unavailable (error_rate=%d%%)", c.errorRate)
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

	if strings.Contains(question, "[SIMULATE_LLM_ERROR]") {
		return "", errors.New("simulated LLM question answering failure")
	}

	if c.errorRate > 0 && rand.Intn(100) < c.errorRate {
		return "", fmt.Errorf("LLM chat service temporarily unavailable (error_rate=%d%%)", c.errorRate)
	}

	return fmt.Sprintf("Ответ на вопрос %q: на основе материалов встречи подтверждаем принятие всех ключевых решений.", question), nil
}
