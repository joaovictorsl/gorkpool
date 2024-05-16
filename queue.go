package main

type Queue[T any] struct {
	elements []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		elements: make([]T, 0, 1_000_000),
	}
}

func (q *Queue[T]) Enqueue(element T) {
	q.elements = append(q.elements, element)
}

func (q *Queue[T]) Dequeue() (T, bool) {
	var zero T
	if len(q.elements) == 0 {
		return zero, false
	}

	element := q.elements[0]
	q.elements = q.elements[1:]
	return element, true
}
