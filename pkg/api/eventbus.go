package api

import (
	"log"
	"sync"
)

// EventBus is the publish/subscribe interface for resource events.
//
// The EventBus is the nervous system of the framework:
// - Storage publishes events when resources change
// - Watch endpoints subscribe and stream to clients
// - Controllers subscribe and process events asynchronously
//
// It enables decoupled, event-driven architecture.
type EventBus interface {
	// Publish sends an event to all subscribers of that resource.
	// Non-blocking - publishers never wait for subscribers.
	Publish(event Event)

	// Subscribe registers a client for events on a specific resource.
	// Returns a Subscription that delivers events through a channel.
	// Multiple subscribers can listen to the same resource simultaneously.
	Subscribe(resource string) *Subscription

	// Unsubscribe removes a subscription.
	// Safe to call multiple times.
	Unsubscribe(subscription *Subscription)

	// Close shuts down the event bus and closes all subscriptions.
	Close() error
}

// SimpleEventBus implements EventBus with goroutines and channels.
//
// Architecture:
// - One goroutine per subscription (drains events from its channel)
// - One publish goroutine per event (fans out to all subscribers)
// - Thread-safe using sync.RWMutex for subscriber management
//
// This design ensures:
// - Slow subscribers don't block publishers or other subscribers
// - Publishers never block
// - Clean shutdown with proper resource cleanup
type SimpleEventBus struct {
	mu           sync.RWMutex
	subscribers  map[string][]*Subscription
	publishQueue chan Event
	done         chan struct{}
	closed       bool
}

// NewEventBus creates a new event bus.
func NewEventBus() EventBus {
	bus := &SimpleEventBus{
		subscribers:  make(map[string][]*Subscription),
		publishQueue: make(chan Event, 1000),
		done:         make(chan struct{}),
	}

	// Start the publisher goroutine
	go bus.publishLoop()

	return bus
}

// Publish enqueues an event for publishing.
// Non-blocking - returns immediately.
// If bus is closed, event is discarded silently.
func (b *SimpleEventBus) Publish(event Event) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}
	b.mu.RUnlock()

	select {
	case b.publishQueue <- event:
	case <-b.done:
		// Bus is closed, discard event silently
	}
}

// Subscribe creates a new subscription for events on a resource.
func (b *SimpleEventBus) Subscribe(resource string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create buffered channel (subscribers should drain quickly, but buffer
	// for brief delays to avoid unnecessary goroutines waiting).
	sendCh := make(chan Event, 100)

	sub := &Subscription{
		Resource: resource,
		Events:   sendCh,
		done:     make(chan struct{}),
		sendCh:   sendCh,
	}

	b.subscribers[resource] = append(b.subscribers[resource], sub)
	log.Printf("Subscribe: %s (now %d watchers)", resource, len(b.subscribers[resource]))

	return sub
}

// Unsubscribe removes a subscription.
func (b *SimpleEventBus) Unsubscribe(sub *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, exists := b.subscribers[sub.Resource]; exists {
		for i, s := range subs {
			if s == sub {
				// Close the send channel
				select {
				case <-sub.done:
				default:
					close(sub.sendCh)
				}

				// Remove from list
				b.subscribers[sub.Resource] = append(subs[:i], subs[i+1:]...)

				log.Printf("Unsubscribe: %s (now %d watchers)", sub.Resource, len(b.subscribers[sub.Resource]))
				return
			}
		}
	}
}

// publishLoop runs in a goroutine and handles event distribution.
// It ensures publishers never block by running distribution in separate goroutines.
func (b *SimpleEventBus) publishLoop() {
	for {
		select {
		case event := <-b.publishQueue:
			// Fan out to subscribers in a separate goroutine
			// This prevents any subscriber from blocking others
			go b.fanOut(event)

		case <-b.done:
			close(b.publishQueue)
			return
		}
	}
}

// fanOut distributes an event to all subscribers of a resource.
// Runs in a separate goroutine per event.
func (b *SimpleEventBus) fanOut(event Event) {
	b.mu.RLock()
	subscribers := b.subscribers[event.Resource]

	// Make a copy of the subscriber list to avoid holding the lock
	// while sending to channels (which could block if subscribers are slow)
	subs := make([]*Subscription, len(subscribers))
	copy(subs, subscribers)
	b.mu.RUnlock()

	// Send to each subscriber
	for _, sub := range subs {
		select {
		case sub.sendCh <- event:
		case <-sub.done:
			// Subscriber closed, skip
		default:
			// Channel full or closed - this shouldn't happen with our buffer,
			// but if it does, we log it and continue (one slow subscriber
			// doesn't block others)
			log.Printf("Event queue full for subscriber: %s", event.Resource)
		}
	}
}

// Close shuts down the event bus.
// It will no longer publish events and closes all subscriptions.
// Safe to call multiple times.
func (b *SimpleEventBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	// Signal done to publishLoop and any waiting Publish calls
	close(b.done)

	// Close all subscriptions
	b.mu.Lock()
	for _, subs := range b.subscribers {
		for _, sub := range subs {
			close(sub.sendCh)
		}
	}
	b.subscribers = make(map[string][]*Subscription)
	b.mu.Unlock()

	return nil
}
