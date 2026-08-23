package workerpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"diplomaMeetHelper/internal/domain"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type JobHandler func(ctx context.Context, job domain.ProcessingJob) error

type WorkerPool struct {
	workers        int
	jobTimeout     time.Duration
	jobQueue       chan domain.ProcessingJob
	handler        JobHandler
	logger         *zap.Logger
	dispatcherDone chan struct{}

	isClosed atomic.Bool
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	eg       *errgroup.Group
}

func New(workers int, queueSize int, jobTimeout time.Duration, handler JobHandler, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(workers)

	return &WorkerPool{
		workers:        workers,
		jobTimeout:     jobTimeout,
		jobQueue:       make(chan domain.ProcessingJob, queueSize),
		handler:        handler,
		logger:         logger,
		dispatcherDone: make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		eg:             eg,
	}
}

func (wp *WorkerPool) Start() {
	go func() {
		defer close(wp.dispatcherDone)
		for {
			select {
			case <-wp.ctx.Done():
				return
			case job, ok := <-wp.jobQueue:
				if !ok {
					return
				}
				currentJob := job
				wp.eg.Go(func() error {
					wp.processJob(currentJob)
					return nil
				})
			}
		}
	}()
}

func (wp *WorkerPool) processJob(job domain.ProcessingJob) {
	jobCtx, jobCancel := context.WithTimeout(wp.ctx, wp.jobTimeout)
	defer jobCancel()

	if err := wp.handler(jobCtx, job); err != nil {
		wp.logger.Error("Job processing failed",
			zap.String("job_id", job.ID.String()),
			zap.String("meeting_id", job.MeetingID.String()),
			zap.Error(err),
		)
	}
}

func (wp *WorkerPool) Submit(job domain.ProcessingJob) error {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

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
	wp.mu.Lock()
	if wp.isClosed.Swap(true) {
		wp.mu.Unlock()
		return
	}
	close(wp.jobQueue)
	wp.mu.Unlock()

	done := make(chan struct{})
	go func() {
		// Wait for dispatcher to finish submitting all queued jobs
		<-wp.dispatcherDone
		// Wait for all running jobs in errgroup to complete
		if err := wp.eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			wp.logger.Warn("Worker pool stopped with error", zap.Error(err))
		}
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
