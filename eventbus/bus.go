package eventbus

import (
	"sync"

	"github.com/ZiplEix/upfluence-test/telemetry"
)

type EventBus[T any] struct {
	mu          sync.RWMutex
	subscribers map[chan T]struct{}
	bufferSize  int
}

// New create a new EventBus
func New[T any](bufferSize int) *EventBus[T] {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &EventBus[T]{
		subscribers: make(map[chan T]struct{}),
		bufferSize:  bufferSize,
	}
}

// Subscribe register a new subscriber and return it's dedicated channel and unsubscribe function
func (b *EventBus[T]) Subscribe() (EventChannel <-chan T, unsubscribe func()) {
	ch := make(chan T, b.bufferSize)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	telemetry.ActiveSubscribers.Add(1)

	unsubscribe = func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, exists := b.subscribers[ch]; exists {
			delete(b.subscribers, ch)
			close(ch)
			telemetry.ActiveSubscribers.Add(-1)
		}
	}

	return ch, unsubscribe
}

// Publish send an event on every channel registered
func (b *EventBus[T]) Publish(event T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// drop event on full channel
			telemetry.DroppedEvents.Add(1)
		}
	}
}

// SubscriberCount return the number of current subscribers
func (b *EventBus[T]) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
