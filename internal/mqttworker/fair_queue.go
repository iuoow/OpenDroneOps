package mqttworker

import (
	"context"
	"sync"
)

// fairQueue is a bounded, per-key round-robin queue. It preserves order for a
// key while ensuring that one hot key cannot consume all scheduling turns.
type fairQueue struct {
	mu               sync.Mutex
	capacity         int
	maxPendingPerKey int
	pending          map[string][]Message
	ready            []string
	scheduled        map[string]bool
	size             int
	closed           bool
	notify           chan struct{}
}

func newFairQueue(capacity, maxPendingPerKey int) *fairQueue {
	return &fairQueue{
		capacity: capacity, maxPendingPerKey: maxPendingPerKey,
		pending: make(map[string][]Message), scheduled: make(map[string]bool),
		notify: make(chan struct{}, 1),
	}
}

func (q *fairQueue) Enqueue(key string, message Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	if q.size >= q.capacity {
		return ErrQueueFull
	}
	if len(q.pending[key]) >= q.maxPendingPerKey {
		return ErrHotKeyBackpressure
	}
	q.pending[key] = append(q.pending[key], message)
	q.size++
	if !q.scheduled[key] {
		q.ready = append(q.ready, key)
		q.scheduled[key] = true
	}
	q.signal()
	return nil
}

func (q *fairQueue) Dequeue(ctx context.Context) (Message, bool) {
	for {
		q.mu.Lock()
		if len(q.ready) > 0 {
			key := q.ready[0]
			q.ready = q.ready[1:]
			messages := q.pending[key]
			message := messages[0]
			messages = messages[1:]
			q.size--
			if len(messages) == 0 {
				delete(q.pending, key)
				delete(q.scheduled, key)
			} else {
				q.pending[key] = messages
				q.ready = append(q.ready, key)
			}
			q.mu.Unlock()
			return message, true
		}
		if q.closed {
			q.mu.Unlock()
			return Message{}, false
		}
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			return Message{}, false
		case <-q.notify:
		}
	}
}

func (q *fairQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.signal()
}

func (q *fairQueue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
