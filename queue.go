/*
Package queue provides a fast, ring-buffer queue based on the version suggested by Dariusz Górecki.
Using this instead of other, simpler, queue implementations (slice+append or linked list) provides
substantial memory and time benefits, and fewer GC pauses.

*/
package queue

import "sync"
import "fmt"

const minQueueLen = 16

// Queue represents a single instance of the queue data structure.
type Queue struct {
	buf               []interface{}
	head, tail, count int
	m                 sync.Mutex
	c                 chan int
	max_len           int
}

// New constructs and returns a new Queue.
func New(max_len int) *Queue {
	return &Queue{
		buf:     make([]interface{}, minQueueLen),
		c:       make(chan int),
		max_len: max_len,
	}
}

// Length returns the number of elements currently stored in the queue.
func (q *Queue) Length() int {
	return q.count
}

// resizes the queue to fit exactly twice its current contents
// this can result in shrinking if the queue is less than half-full
func (q *Queue) resize() {
	newBuf := make([]interface{}, q.count*2)

	if q.tail > q.head {
		copy(newBuf, q.buf[q.head:q.tail])
	} else {
		n := copy(newBuf, q.buf[q.head:])
		copy(newBuf[n:], q.buf[:q.tail])
	}

	q.head = 0
	q.tail = q.count
	q.buf = newBuf
}

// Add puts an element on the end of the queue.
func (q *Queue) Add(elem interface{}) {
	q.m.Lock()
	defer q.m.Unlock()

	if q.count >= q.max_len {
		return
	}
	if q.count == len(q.buf) {
		q.resize()
	}

	q.buf[q.tail] = elem
	q.tail = (q.tail + 1) % len(q.buf)
	q.count++
	go func() {
		q.c <- 1
	}()
}

// Peek returns the element at the head of the queue. This call panics
// if the queue is empty.
func (q *Queue) Peek() interface{} {
	q.m.Lock()
	defer q.m.Unlock()

	if q.count <= 0 {
		panic("queue: Peek() called on empty queue")
	}
	return q.buf[q.head]
}

// Get returns the element at index i in the queue. If the index is
// invalid, the call will panic.
func (q *Queue) Get(i int) interface{} {
	q.m.Lock()
	defer q.m.Unlock()

	if i < 0 || i >= q.count {
		panic("queue: Get() called with index out of range")
	}
	return q.buf[(q.head+i)%len(q.buf)]
}

// Remove removes the element from the front of the queue. If you actually
// want the element, call Peek first. This call panics if the queue is empty.
func (q *Queue) Remove() {
	q.m.Lock()
	defer q.m.Unlock()

	if q.count <= 0 {
		panic("queue: Remove() called on empty queue")
	}
	q.buf[q.head] = nil
	q.head = (q.head + 1) % len(q.buf)
	q.count--
	if len(q.buf) > minQueueLen && q.count*4 == len(q.buf) {
		q.resize()
	}
}

func (q *Queue) PeekAndRemove() interface{} {
	q.m.Lock()
	defer q.m.Unlock()

	if q.count <= 0 {
		panic("queue: Remove() called on empty queue")
	}
	h := q.buf[q.head]
	q.buf[q.head] = nil
	q.head = (q.head + 1) % len(q.buf)
	q.count--
	if len(q.buf) > minQueueLen && q.count*4 == len(q.buf) {
		q.resize()
	}
	return h
}

func (q *Queue) Wait() error {
	x, ok := <-q.c
	if ok && x == 1 {
		return nil
	}
	return fmt.Errorf("Queue Stopped")
}

func (q *Queue) Stop() {
	q.c <- 0
	close(q.c)
}
