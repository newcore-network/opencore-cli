package builder

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool(4)

	if pool == nil {
		t.Fatal("NewWorkerPool returned nil")
	}

	if pool.workers != 4 {
		t.Errorf("Expected 4 workers, got %d", pool.workers)
	}

	if pool.taskChan == nil {
		t.Error("Task channel is nil")
	}

	if pool.resultChan == nil {
		t.Error("Result channel is nil")
	}

	if pool.ctx == nil {
		t.Error("Context is nil")
	}

	if pool.cancel == nil {
		t.Error("Cancel function is nil")
	}
}

func TestWorkerPoolBasicExecution(t *testing.T) {
	pool := NewWorkerPool(2)

	// Simple build function that just returns success
	buildFunc := func(task BuildTask) BuildResult {
		return BuildResult{
			Task:     task,
			Success:  true,
			Duration: 10 * time.Millisecond,
		}
	}

	pool.Start(buildFunc)

	// Submit a single task
	task := BuildTask{
		Path:         "./core",
		ResourceName: "[core]",
		Type:         TypeCore,
	}

	pool.Submit(task)

	// Get result
	select {
	case result := <-pool.Results():
		if !result.Success {
			t.Error("Expected successful result")
		}
		if result.Task.ResourceName != "[core]" {
			t.Errorf("Expected resource name '[core]', got '%s'", result.Task.ResourceName)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for result")
	}

	pool.Close()
}

func TestWorkerPoolMultipleTasks(t *testing.T) {
	pool := NewWorkerPool(4)

	var processedCount int32

	buildFunc := func(task BuildTask) BuildResult {
		atomic.AddInt32(&processedCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
		return BuildResult{
			Task:    task,
			Success: true,
		}
	}

	pool.Start(buildFunc)

	tasks := []BuildTask{
		{Path: "./core", ResourceName: "[core]", Type: TypeCore},
		{Path: "./resources/admin", ResourceName: "admin", Type: TypeResource},
		{Path: "./resources/inventory", ResourceName: "inventory", Type: TypeResource},
		{Path: "./standalone/utils", ResourceName: "utils", Type: TypeStandalone},
	}

	pool.SubmitAll(tasks)

	// Collect all results
	results := make([]BuildResult, 0, len(tasks))
	for i := 0; i < len(tasks); i++ {
		select {
		case result := <-pool.Results():
			results = append(results, result)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for result %d", i)
		}
	}

	pool.Close()

	if len(results) != len(tasks) {
		t.Errorf("Expected %d results, got %d", len(tasks), len(results))
	}

	if int(processedCount) != len(tasks) {
		t.Errorf("Expected %d processed tasks, got %d", len(tasks), processedCount)
	}

	// Verify all tasks were successful
	for _, r := range results {
		if !r.Success {
			t.Errorf("Task %s failed", r.Task.ResourceName)
		}
	}
}

func TestWorkerPoolParallelism(t *testing.T) {
	workers := 4
	pool := NewWorkerPool(workers)

	var maxConcurrent int32
	var currentConcurrent int32

	buildFunc := func(task BuildTask) BuildResult {
		current := atomic.AddInt32(&currentConcurrent, 1)

		// Track max concurrent executions
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if current <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond) // Long enough to have overlap

		atomic.AddInt32(&currentConcurrent, -1)

		return BuildResult{
			Task:    task,
			Success: true,
		}
	}

	pool.Start(buildFunc)

	// Submit more tasks than workers
	tasks := make([]BuildTask, 8)
	for i := 0; i < 8; i++ {
		tasks[i] = BuildTask{
			Path:         "./test",
			ResourceName: "test",
			Type:         TypeResource,
		}
	}

	pool.SubmitAll(tasks)

	// Collect results
	for i := 0; i < len(tasks); i++ {
		<-pool.Results()
	}

	pool.Close()

	// Verify parallelism happened (at least 2 concurrent)
	if maxConcurrent < 2 {
		t.Errorf("Expected at least 2 concurrent workers, got %d", maxConcurrent)
	}

	// Should not exceed worker count
	if maxConcurrent > int32(workers) {
		t.Errorf("Max concurrent %d exceeded worker count %d", maxConcurrent, workers)
	}
}

func TestWorkerPoolCancel(t *testing.T) {
	pool := NewWorkerPool(2)

	var taskStarted int32
	buildFunc := func(task BuildTask) BuildResult {
		atomic.AddInt32(&taskStarted, 1)
		// Check context periodically to allow cancel to interrupt
		for i := 0; i < 20; i++ {
			select {
			case <-pool.ctx.Done():
				return BuildResult{Task: task, Success: false}
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
		return BuildResult{Task: task, Success: true}
	}

	pool.Start(buildFunc)

	// Submit a task
	pool.Submit(BuildTask{
		Path:         "./core",
		ResourceName: "[core]",
		Type:         TypeCore,
	})

	// Wait a bit for task to start
	time.Sleep(100 * time.Millisecond)

	// Verify task started
	if atomic.LoadInt32(&taskStarted) == 0 {
		t.Error("Task should have started")
	}

	// Cancel - this should interrupt the task
	pool.Cancel()

	// Close the pool gracefully
	pool.Close()

	// The test passes if we get here without hanging
	// (the worker should have been interrupted by cancel)
}

func TestWorkerPoolCollectResults(t *testing.T) {
	pool := NewWorkerPool(2)

	var callCount int32

	buildFunc := func(task BuildTask) BuildResult {
		count := atomic.AddInt32(&callCount, 1)
		// Alternate between success and failure
		success := count%2 == 1
		return BuildResult{
			Task:    task,
			Success: success,
		}
	}

	pool.Start(buildFunc)

	// Submit 6 tasks
	for i := 0; i < 6; i++ {
		pool.Submit(BuildTask{
			Path:         "./test",
			ResourceName: "test",
			Type:         TypeResource,
		})
	}

	// Collect results
	results, successCount, failCount := pool.CollectResults(6)

	pool.Close()

	if len(results) != 6 {
		t.Errorf("Expected 6 results, got %d", len(results))
	}

	// With alternating success/fail, should have 3 each
	if successCount != 3 {
		t.Errorf("Expected 3 successes, got %d", successCount)
	}

	if failCount != 3 {
		t.Errorf("Expected 3 failures, got %d", failCount)
	}
}

func TestWorkerPoolWithCustomCompiler(t *testing.T) {
	pool := NewWorkerPool(2)

	var receivedCompiler string

	buildFunc := func(task BuildTask) BuildResult {
		receivedCompiler = task.CustomCompiler
		return BuildResult{
			Task:    task,
			Success: true,
		}
	}

	pool.Start(buildFunc)

	task := BuildTask{
		Path:           "./core",
		ResourceName:   "[core]",
		Type:           TypeCore,
		CustomCompiler: "./scripts/custom-build.js",
	}

	pool.Submit(task)

	<-pool.Results()
	pool.Close()

	if receivedCompiler != "./scripts/custom-build.js" {
		t.Errorf("Expected custom compiler './scripts/custom-build.js', got '%s'", receivedCompiler)
	}
}

func TestWorkerPoolEmptyTasks(t *testing.T) {
	pool := NewWorkerPool(2)

	buildFunc := func(task BuildTask) BuildResult {
		return BuildResult{Task: task, Success: true}
	}

	pool.Start(buildFunc)

	// Submit empty slice
	pool.SubmitAll([]BuildTask{})

	// Close immediately - should not hang
	done := make(chan struct{})
	go func() {
		pool.Close()
		close(done)
	}()

	select {
	case <-done:
		// Good
	case <-time.After(1 * time.Second):
		t.Error("Pool close hung with empty tasks")
	}
}

func TestWorkerPoolResourceTypes(t *testing.T) {
	pool := NewWorkerPool(2)

	receivedTypes := make(map[ResourceType]bool)
	var mu sync.Mutex

	buildFunc := func(task BuildTask) BuildResult {
		mu.Lock()
		receivedTypes[task.Type] = true
		mu.Unlock()
		return BuildResult{Task: task, Success: true}
	}

	pool.Start(buildFunc)

	tasks := []BuildTask{
		{Type: TypeCore},
		{Type: TypeResource},
		{Type: TypeStandalone},
		{Type: TypeViews},
		{Type: TypeCopy},
	}

	pool.SubmitAll(tasks)

	for range tasks {
		<-pool.Results()
	}

	pool.Close()

	// Verify all types were processed
	expectedTypes := []ResourceType{TypeCore, TypeResource, TypeStandalone, TypeViews, TypeCopy}
	for _, rt := range expectedTypes {
		if !receivedTypes[rt] {
			t.Errorf("Resource type %s was not processed", rt)
		}
	}
}
