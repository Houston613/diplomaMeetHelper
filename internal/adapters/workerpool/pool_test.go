package workerpool_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"diplomaMeetHelper/internal/adapters/workerpool"
	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestWorkerPool_ConcurrentProcessing(t *testing.T) {
	var processed atomic.Int32
	totalJobs := 20

	handler := func(ctx context.Context, job domain.ProcessingJob) error {
		time.Sleep(10 * time.Millisecond)
		processed.Add(1)
		return nil
	}

	pool := workerpool.New(3, 50, 2*time.Second, handler, zap.NewNop())
	pool.Start()

	for i := 0; i < totalJobs; i++ {
		job := domain.ProcessingJob{
			ID:        uuid.New(),
			MeetingID: uuid.New(),
		}
		if err := pool.Submit(job); err != nil {
			t.Fatalf("failed to submit job: %v", err)
		}
	}

	pool.Stop(2 * time.Second)

	if int(processed.Load()) != totalJobs {
		t.Errorf("expected %d jobs processed, got %d", totalJobs, processed.Load())
	}
}
