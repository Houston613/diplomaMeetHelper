package workerpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"diplomaMeetHelper/internal/domain"

	"go.uber.org/zap"
)

type JobHandler func(ctx context.Context, job domain.ProcessingJob) error

type WorkerPool struct {
	workers    int
	jobTimeout time.Duration
	jobQueue   chan domain.ProcessingJob
	handler    JobHandler
	logger     *zap.Logger
	wg       sync.WaitGroup
	isClosed atomic.Bool
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(workers int, queueSize int, jobTimeout time.Duration, handler JobHandler, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:    workers,
		jobTimeout: jobTimeout,
		jobQueue:   make(chan domain.ProcessingJob, queueSize),
		handler:    handler,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (wp *WorkerPool) Start() {
	for i := 1; i <= wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for job := range wp.jobQueue {
		wp.processJob(id, job)
	}
}

func (wp *WorkerPool) processJob(workerID int, job domain.ProcessingJob) {
	jobCtx, jobCancel := context.WithTimeout(wp.ctx, wp.jobTimeout)
	defer jobCancel()

	if err := wp.handler(jobCtx, job); err != nil {
		wp.logger.Error("Job processing failed",
			zap.Int("worker_id", workerID),
			zap.String("job_id", job.ID.String()),
			zap.String("meeting_id", job.MeetingID.String()),
			zap.Error(err),
		)
	}
}

func (wp *WorkerPool) Submit(job domain.ProcessingJob) error {
	if wp.isClosed.Load() {
		return errors.New("worker pool is stopped")
	}

	select {
	case wp.jobQueue <- job:
		return nil
	default:
		return domain.ErrQueueFull
	}
}

func (wp *WorkerPool) Stop(gracefulTimeout time.Duration) {
	wp.isClosed.Store(true)
	close(wp.jobQueue)

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.cancel()
	case <-time.After(gracefulTimeout):
		wp.cancel()
		<-done
	}
}
