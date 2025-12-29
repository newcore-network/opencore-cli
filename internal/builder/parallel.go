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
	buildFunc  func(BuildTask) BuildResult
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:    workers,
		taskChan:   make(chan BuildTask, 100),
		resultChan: make(chan BuildResult, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins the worker pool with the given build function
func (wp *WorkerPool) Start(buildFunc func(BuildTask) BuildResult) {
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
			result := wp.buildFunc(task)
			select {
			case wp.resultChan <- result:
			case <-wp.ctx.Done():
				return
			}
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
	close(wp.taskChan)
	wp.wg.Wait()
	close(wp.resultChan)
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
