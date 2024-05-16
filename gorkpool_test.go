package gorkpool_test

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/joaovictorsl/gorkpool"
)

type testWorker struct {
	id     int
	input  chan int
	output chan int
	done   chan struct{}
}

func newTestWorker(id int, input chan int, output chan int) *testWorker {
	return &testWorker{
		id:     id,
		input:  input,
		output: output,
		done:   make(chan struct{}),
	}
}

func (w *testWorker) ID() int {
	return w.id
}

func (w *testWorker) Process() {
	for {
		select {
		case <-w.done:
			return
		case x, ok := <-w.input:
			if !ok {
				return
			}
			w.output <- -x
		}
	}
}

func (w *testWorker) SignalRemoval() {
	w.done <- struct{}{}
}

func setupPool() (*gorkpool.GorkPool[int, int, int], context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	inputCh := make(chan int, 10)
	outputCh := make(chan int, 10)
	return gorkpool.NewGorkPool(ctx, inputCh, outputCh, func(id int, ic chan int, oc chan int) (gorkpool.GorkWorker[int, int, int], error) {
		return newTestWorker(id, inputCh, outputCh), nil
	}), cancel
}

func TestAddWorker(t *testing.T) {
	// Setup
	pool, cancel := setupPool()
	// Assert
	if pool.Length() != 0 {
		t.Errorf("expected pool to be empty, got %d", pool.Length())
	}
	// Action
	pool.AddWorker(0)
	// Assert
	if pool.Length() != 1 {
		t.Errorf("expected pool to have %d worker(s), got %d", 1, pool.Length())
	} else if !pool.Contains(0) {
		t.Error("expected worker 0 to be in pool but wasn't")
	}
	// Cleanup
	cancel()
	<-pool.OutputCh()
}

func TestAddWorkerDuplicatedId(t *testing.T) {
	// Setup
	pool, cancel := setupPool()
	expectedErr := gorkpool.NewErrIdConflict(0)
	// Action
	pool.AddWorker(0)
	err := pool.AddWorker(0)
	// Assert
	if err == nil {
		t.Error("expected error when adding duplicated id, got nil")
	} else if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got %v", expectedErr, err)
	}
	// Cleanup
	cancel()
	<-pool.OutputCh()
}

func TestRemoveWorker(t *testing.T) {
	// Setup
	pool, cancel := setupPool()
	pool.AddWorker(0)
	pool.AddWorker(1)
	// Action
	pool.RemoveWorker()
	// Assert
	if pool.Length() != 1 {
		t.Errorf("expected pool to have %d worker(s), got %d", 1, pool.Length())
	}
	// Cleanup
	cancel()
	<-pool.OutputCh()
}

func TestRemoveWorkerEmpty(t *testing.T) {
	// Setup
	pool, cancel := setupPool()
	// Action
	pool.RemoveWorker()
	// Assert
	if pool.Length() != 0 {
		t.Errorf("expected pool to have %d worker(s), got %d", 0, pool.Length())
	}
	// Cleanup
	cancel()
	<-pool.OutputCh()
}

func TestRemoveWorkerById(t *testing.T) {
	// Setup
	pool, cancel := setupPool()
	for i := 0; i < 10; i++ {
		pool.AddWorker(i)
	}
	target := 7
	// Action
	pool.RemoveWorkerById(target)
	// Assert
	for i := 0; i < 10; i++ {
		if i != target && !pool.Contains(i) {
			t.Errorf("expected pool to contain worker %d, but it didn't", i)
		} else if i == target && pool.Contains(i) {
			t.Errorf("expected pool to not contain worker %d, but it did", i)
		}
	}
	if pool.Length() != 9 {
		t.Errorf("expected pool to have %d worker(s), got %d", 9, pool.Length())
	}
	// Cleanup
	cancel()
	<-pool.OutputCh()
}

func TestGracefullyShutdown(t *testing.T) {
	// Setup
	pool, cancel := setupPool()
	for i := 0; i < 10; i++ {
		pool.AddWorker(i)
	}
	runningGoroutines := runtime.NumGoroutine() - 1 // Removing golang test runner's goroutine
	// Assert
	if runningGoroutines != 12 { // 10 Workers, 1 pool and my test's goroutine
		t.Errorf("expected 12 goroutines to be running, got %d", runningGoroutines)
	}
	// Action
	cancel()
	<-pool.OutputCh()                              // Wait for other goroutines to end
	runningGoroutines = runtime.NumGoroutine() - 1 // Removing golang test runner's goroutine
	// Assert
	if runningGoroutines != 1 {
		t.Errorf("expected 1 goroutine to be running, got %d", runningGoroutines)
	}
}
