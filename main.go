package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	results := make([]float64, 4)
	for i := 1; i <= 4; i++ {
		for j := 0; j < 10; j++ {
			throughput := runRound(uint(i))
			results[i-1] += throughput
		}

		fmt.Printf("Worker count: %d, average throughput: %.2f tasks/ms\n", i, results[i-1]/10)
	}

}

func runRound(workerCount uint) float64 {
	id := 0
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewGorkPool[[]int](ctx, workerCount, func() GorkWorker[[]int] {
		id++
		return &SortWorker{
			workerId: id,
			workCh:   make(chan []int),
		}
	})
	pool.Start()

	start := time.Now()
	i := 0
	for ; i < 1_000_000; i++ {
		ok := pool.Enqueue([]int{658, 345, 623, 667, 23, 4, 1, 22, 3, 56, 2})
		if !ok {
			panic("pool is terminating")
		}
	}

	cancel()
	pool.Wait()
	elapsed := time.Since(start)

	return float64(i) / float64(elapsed.Milliseconds())
}
