package mock

import (
	"context"
	"fmt"
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
		return strings.TrimSpace(string(data)), nil
	}

	return fmt.Sprintf("Транскрипция аудиозаписи %s: Здесь просто текст и ничего интересного пока что.", filePath), nil
}
