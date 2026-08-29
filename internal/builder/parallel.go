package builder

import (
	"context"
	"sync"
)

// WorkerPool manages parallel build workers
type WorkerPool struct {
	workers    int
	taskChan   chan BuildTask
	resultChan chan BuildResult
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	buildFunc  func(context.Context, BuildTask) BuildResult
	closeOnce  sync.Once
	resultMu   sync.Mutex
	resultCond *sync.Cond
	pending    []BuildResult
	finished   bool
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		workers:    workers,
		taskChan:   make(chan BuildTask, 100),
		resultChan: make(chan BuildResult),
		ctx:        ctx,
		cancel:     cancel,
	}
	pool.resultCond = sync.NewCond(&pool.resultMu)
	go pool.dispatchResults()
	return pool
}

// Start begins the worker pool with the given build function.
func (wp *WorkerPool) Start(buildFunc func(BuildTask) BuildResult) {
	wp.StartWithContext(func(_ context.Context, task BuildTask) BuildResult {
		return buildFunc(task)
	})
}

// StartWithContext begins the worker pool with a cancellation-aware build function.
func (wp *WorkerPool) StartWithContext(buildFunc func(context.Context, BuildTask) BuildResult) {
	wp.buildFunc = buildFunc

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// worker is the goroutine that processes build tasks
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskChan:
			if !ok {
				return
			}
			result := wp.buildFunc(wp.ctx, task)
			wp.enqueueResult(result)
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) enqueueResult(result BuildResult) {
	wp.resultMu.Lock()
	wp.pending = append(wp.pending, result)
	wp.resultCond.Signal()
	wp.resultMu.Unlock()
}

func (wp *WorkerPool) dispatchResults() {
	defer close(wp.resultChan)
	for {
		wp.resultMu.Lock()
		for len(wp.pending) == 0 && !wp.finished {
			wp.resultCond.Wait()
		}
		if len(wp.pending) == 0 && wp.finished {
			wp.resultMu.Unlock()
			return
		}
		result := wp.pending[0]
		wp.pending[0] = BuildResult{}
		wp.pending = wp.pending[1:]
		wp.resultMu.Unlock()
		select {
		case wp.resultChan <- result:
		case <-wp.ctx.Done():
			return
		}
	}
}

// Submit adds a task to the pool
func (wp *WorkerPool) Submit(task BuildTask) {
	select {
	case wp.taskChan <- task:
	case <-wp.ctx.Done():
	}
}

// SubmitAll adds multiple tasks to the pool
func (wp *WorkerPool) SubmitAll(tasks []BuildTask) {
	for _, task := range tasks {
		wp.Submit(task)
	}
}

// Results returns the results channel for receiving build results
func (wp *WorkerPool) Results() <-chan BuildResult {
	return wp.resultChan
}

// Close shuts down the worker pool gracefully
// Call this after all tasks have been submitted
func (wp *WorkerPool) Close() {
	wp.closeOnce.Do(func() {
		close(wp.taskChan)
		wp.wg.Wait()
		wp.resultMu.Lock()
		wp.finished = true
		wp.resultCond.Broadcast()
		wp.resultMu.Unlock()
	})
}

// Cancel cancels all workers immediately
func (wp *WorkerPool) Cancel() {
	wp.cancel()
}

// Wait blocks until all submitted tasks are processed
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// CollectResults collects all results until the pool is closed
// Returns slice of results and counts of success/failure
func (wp *WorkerPool) CollectResults(total int) ([]BuildResult, int, int) {
	results := make([]BuildResult, 0, total)
	successCount := 0
	failCount := 0

	for i := 0; i < total; i++ {
		result := <-wp.resultChan
		results = append(results, result)
		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}

	return results, successCount, failCount
}
