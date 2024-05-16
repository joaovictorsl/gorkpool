package gorkpool

import (
	"context"
	"sync"
)

type GorkPool[Id comparable, Task any, Result any] struct {
	mutex          *sync.Mutex
	workers        map[Id]GorkWorker[Id, Task, Result]
	createWorkerFn WorkerFactoryFn[Id, Task, Result]

	wg       *sync.WaitGroup
	ctx      context.Context
	inputCh  chan Task
	outputCh chan Result
}

type GorkWorker[Id comparable, Task any, Result any] interface {
	ID() Id
	Process()
	SignalRemoval()
}

type WorkerFactoryFn[Id comparable, Task any, Result any] func(Id, chan Task, chan Result) (GorkWorker[Id, Task, Result], error)

func NewGorkPool[Id comparable, Task any, Result any](
	ctx context.Context,
	inputCh chan Task,
	outputCh chan Result,
	createWorkerFn WorkerFactoryFn[Id, Task, Result],
) *GorkPool[Id, Task, Result] {
	pool := &GorkPool[Id, Task, Result]{
		mutex:          &sync.Mutex{},
		workers:        make(map[Id]GorkWorker[Id, Task, Result], 0),
		createWorkerFn: createWorkerFn,
		wg:             &sync.WaitGroup{},
		ctx:            ctx,
		inputCh:        inputCh,
		outputCh:       outputCh,
	}

	go pool.gracefullyShutdown()

	return pool
}

func (p *GorkPool[Id, Task, Result]) AddWorker(id Id) error {
	w, err := p.createWorkerFn(id, p.inputCh, p.outputCh)
	if err != nil {
		return err
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()
	if _, ok := p.workers[w.ID()]; ok {
		return NewErrIdConflict(w.ID())
	}

	p.wg.Add(1)
	p.workers[w.ID()] = w
	go func(w GorkWorker[Id, Task, Result]) {
		w.Process()
		p.wg.Done()
	}(w)

	return nil
}

func (p *GorkPool[Id, Task, Result]) RemoveWorker() GorkWorker[Id, Task, Result] {
	p.mutex.Lock()

	// Removes the first one on the iteration
	var target GorkWorker[Id, Task, Result]
	for id, w := range p.workers {
		target = w
		delete(p.workers, id)
		break
	}
	p.mutex.Unlock()

	// If no one was removed
	if target == nil {
		return nil
	}

	target.SignalRemoval()
	return target
}

func (p *GorkPool[Id, Task, Result]) RemoveWorkerById(id Id) GorkWorker[Id, Task, Result] {
	p.mutex.Lock()
	target, ok := p.workers[id]
	if !ok {
		p.mutex.Unlock()
		return nil
	}

	delete(p.workers, id)
	p.mutex.Unlock()

	target.SignalRemoval()
	return target
}

func (p *GorkPool[Id, Task, Result]) Length() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.workers)
}

func (p *GorkPool[Id, Task, Result]) Contains(id Id) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_, ok := p.workers[id]
	return ok
}

func (p *GorkPool[Id, Task, Result]) AddTask(task Task) {
	p.inputCh <- task
}

func (p *GorkPool[Id, Task, Result]) OutputCh() chan Result {
	return p.outputCh
}

func (p *GorkPool[Id, Task, Result]) gracefullyShutdown() {
	<-p.ctx.Done()
	close(p.inputCh)  // Stop receiving new tasks
	p.wg.Wait()       // Wait all workers to finish
	close(p.outputCh) // Indicate that this gorkpool is done
}
