package main

import (
	"context"
	"sync"
)

type GorkPool[T any] struct {
	workers      []GorkWorker[T]
	workerFabric func() GorkWorker[T]

	terminating bool
	done        chan struct{}
	mutex       *sync.RWMutex
	ctx         context.Context

	i int
}

type GorkWorker[T any] interface {
	Work()
	GetWorkChannel() chan<- T
}

func NewGorkPool[T any](ctx context.Context, workerCount uint, workerFabric func() GorkWorker[T]) *GorkPool[T] {
	pool := &GorkPool[T]{
		workers:      make([]GorkWorker[T], workerCount),
		workerFabric: workerFabric,
		terminating:  false,
		done:         make(chan struct{}),
		mutex:        &sync.RWMutex{},
		ctx:          ctx,
	}

	for i := uint(0); i < workerCount; i++ {
		pool.workers[i] = workerFabric()
	}

	return pool
}

func (p *GorkPool[T]) Enqueue(input T) bool {
	// p.mutex.Lock()
	// defer p.mutex.Unlock()

	if p.terminating {
		return false
	}

	p.workers[p.i].GetWorkChannel() <- input
	p.i = (p.i + 1) % len(p.workers)
	return true
}

func (p *GorkPool[T]) Wait() {
	<-p.done
}

func (p *GorkPool[T]) Start() {
	for _, worker := range p.workers {
		go worker.Work()
	}
	// Serve workers
	go func() {
		<-p.ctx.Done()
		p.gracefullyShutdown()
	}()
}

func (p *GorkPool[T]) gracefullyShutdown() {
	// p.mutex.Lock()
	p.terminating = true
	// p.mutex.Unlock()

	for _, worker := range p.workers {
		close(worker.GetWorkChannel())
	}

	p.done <- struct{}{}
}
