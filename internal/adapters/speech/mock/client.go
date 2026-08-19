package mock

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type SpeechClient struct {
	delay     time.Duration
	errorRate int
}

func NewSpeechClient(delay time.Duration, errorRate int) *SpeechClient {
	return &SpeechClient{
		delay:     delay,
		errorRate: errorRate,
	}
}

func (c *SpeechClient) Transcribe(ctx context.Context, filePath string) (string, error) {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	data, err := os.ReadFile(filePath)
	if err == nil && len(data) > 0 {
		content := strings.TrimSpace(string(data))
		if strings.Contains(content, "[SIMULATE_SPEECH_ERROR]") {
			return "", errors.New("simulated speech recognition failure")
		}
		return content, nil
	}

	if c.errorRate > 0 && rand.Intn(100) < c.errorRate {
		return "", fmt.Errorf("speech recognition service temporarily unavailable (error_rate=%d%%)", c.errorRate)
	}

	return fmt.Sprintf("Транскрипция аудиозаписи %s: Обсуждение архитектуры проекта, этапов интеграции и критериев защиты диплома.", filePath), nil
}
