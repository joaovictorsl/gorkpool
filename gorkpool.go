package gorkpool

import (
	"context"
	"sync"
)

type GorkPool[T any] struct {
	workers      []GorkWorker[T]
	workerFabric func(<-chan T) GorkWorker[T]

	done   chan struct{}
	wg     *sync.WaitGroup
	ctx    context.Context
	taskCh chan T
}

type GorkWorker[T any] interface {
	Process()
}

func NewGorkPool[T any](ctx context.Context, workerCount uint, workerFabric func(<-chan T) GorkWorker[T]) *GorkPool[T] {
	pool := &GorkPool[T]{
		workers:      make([]GorkWorker[T], workerCount),
		workerFabric: workerFabric,
		done:         make(chan struct{}),
		wg:           &sync.WaitGroup{},
		ctx:          ctx,
		taskCh:       make(chan T, workerCount),
	}

	for i := uint(0); i < workerCount; i++ {
		pool.workers[i] = workerFabric(pool.taskCh)
	}

	pool.start()

	return pool
}

func (p *GorkPool[T]) AddTask(task T) {
	p.taskCh <- task
}

func (p *GorkPool[T]) Wait() {
	<-p.done
}

func (p *GorkPool[T]) start() {
	p.wg.Add(len(p.workers))

	for _, worker := range p.workers {
		go func(w GorkWorker[T]) {
			w.Process()
			p.wg.Done()
		}(worker)
	}

	go p.gracefullyShutdown()
}

func (p *GorkPool[T]) gracefullyShutdown() {
	<-p.ctx.Done()
	close(p.taskCh) // Stop receiving new tasks
	p.wg.Wait()     // Wait all workers to finish
	close(p.done)   // Indicate that this gorkpool is done
}
