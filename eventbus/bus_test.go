package eventbus

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Custom buffer size > 0
	bus := New[int](50)
	if bus == nil {
		t.Fatal("expected non-nil EventBus")
	}
	ch, unsub := bus.Subscribe()
	defer unsub()
	if cap(ch) != 50 {
		t.Errorf("expected channel cap 50, got %d", cap(ch))
	}

	// Default buffer size when <= 0
	busZero := New[string](0)
	chZero, unsubZero := busZero.Subscribe()
	defer unsubZero()
	if cap(chZero) != 100 {
		t.Errorf("expected default channel cap 100 for 0, got %d", cap(chZero))
	}

	busNegative := New[string](-5)
	chNeg, unsubNeg := busNegative.Subscribe()
	defer unsubNeg()
	if cap(chNeg) != 100 {
		t.Errorf("expected default channel cap 100 for negative, got %d", cap(chNeg))
	}
}

func TestSubscribeAndUnsubscribe(t *testing.T) {
	bus := New[string](10)

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected initial subscriber count 0, got %d", bus.SubscriberCount())
	}

	ch, unsub := bus.Subscribe()
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected subscriber count 1, got %d", bus.SubscriberCount())
	}

	// First unsubscribe closes channel and removes subscriber
	unsub()
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected subscriber count 0 after unsub, got %d", bus.SubscriberCount())
	}

	// Check channel is closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed, but received value")
		}
	default:
		t.Error("expected channel to be closed immediately")
	}

	// Second unsubscribe should be idempotent and not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("second unsub panicked: %v", r)
		}
	}()
	unsub()
}

func TestPublish(t *testing.T) {
	bus := New[int](10)

	ch1, unsub1 := bus.Subscribe()
	defer unsub1()
	ch2, unsub2 := bus.Subscribe()
	defer unsub2()

	bus.Publish(42)
	bus.Publish(100)

	// Verify ch1
	val1 := <-ch1
	val2 := <-ch1
	if val1 != 42 || val2 != 100 {
		t.Errorf("ch1 received (%d, %d), expected (42, 100)", val1, val2)
	}

	// Verify ch2
	val1 = <-ch2
	val2 = <-ch2
	if val1 != 42 || val2 != 100 {
		t.Errorf("ch2 received (%d, %d), expected (42, 100)", val1, val2)
	}
}

func TestPublish_DropOnFullChannel(t *testing.T) {
	bus := New[int](1)

	ch, unsub := bus.Subscribe()
	defer unsub()

	// Fill the buffer of size 1
	bus.Publish(1)
	// Second publish will hit the default case and drop the event
	bus.Publish(2)
	// Third publish will also drop
	bus.Publish(3)

	val := <-ch
	if val != 1 {
		t.Errorf("expected first event 1, got %d", val)
	}

	// Ensure no more events are in the channel
	select {
	case v := <-ch:
		t.Errorf("expected channel to be empty, got %d", v)
	default:
		// Success: event 2 and 3 were dropped
	}
}

func TestSubscriberCount(t *testing.T) {
	bus := New[int](10)

	if count := bus.SubscriberCount(); count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	_, unsub1 := bus.Subscribe()
	_, unsub2 := bus.Subscribe()
	_, unsub3 := bus.Subscribe()

	if count := bus.SubscriberCount(); count != 3 {
		t.Errorf("expected 3 subscribers, got %d", count)
	}

	unsub1()
	if count := bus.SubscriberCount(); count != 2 {
		t.Errorf("expected 2 subscribers after 1 unsub, got %d", count)
	}

	unsub2()
	unsub3()
	if count := bus.SubscriberCount(); count != 0 {
		t.Errorf("expected 0 subscribers, got %d", count)
	}
}

func TestEventBus_Concurrency(t *testing.T) {
	bus := New[int](50)
	var wg sync.WaitGroup

	const numSubscribers = 10
	const numPublishers = 10
	const numMessages = 100

	stop := make(chan struct{})

	// Launch concurrent subscribers
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, unsub := bus.Subscribe()
			defer unsub()

			for {
				select {
				case <-stop:
					return
				case _, ok := <-ch:
					if !ok {
						return
					}
				}
			}
		}()
	}

	// Launch concurrent publishers
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(pubID int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				bus.Publish(pubID*1000 + j)
				time.Sleep(10 * time.Microsecond)
			}
		}(i)
	}

	// Concurrent subscribe and unsub
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, unsub := bus.Subscribe()
				_ = bus.SubscriberCount()
				time.Sleep(50 * time.Microsecond)
				unsub()
			}
		}()
	}

	// Give publishers time to finish, then close stop
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}
