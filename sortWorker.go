package main

import (
	"golang.org/x/exp/slices"
)

type SortWorker struct {
	workerId int
	workCh   chan []int
}

func (w *SortWorker) GetWorkChannel() chan<- []int {
	return w.workCh
}

func (w *SortWorker) Work() {
	for {
		data, ok := <-w.workCh
		if !ok {
			return
		}
		slices.Sort(data)
	}
}
