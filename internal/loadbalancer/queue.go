package loadbalancer

import (
	"container/list"
	"fmt"
)

type Queue struct {
	queue *list.List
}

func NewQueue() *Queue {
	return &Queue{queue: list.New()}
}

func (q *Queue) Enqueue(value *Request) {
	q.queue.PushBack(value)
}

func (q *Queue) Dequeue() (*Request, error) {
	if q.queue.Len() > 0 {
		el := q.queue.Front()
		q.queue.Remove(el)
		return el.Value.(*Request), nil
	}
	return nil, fmt.Errorf("pop error: Queue is empty")
}

func (q *Queue) IsEmpty() bool {
	return q.queue.Len() == 0
}
