// pkg/api/event.go
//
// This file defines the structures and interfaces for managing events in the
// dynamic API server. Events represent changes to resources (e.g., creation,
// modification, deletion) and are used to notify subscribers (watchers) of
// these changes. The EventBus interface provides a mechanism for publishing
// events and managing subscriptions.

package api

import "time"

// EventType represents the type of event that occurred.
type EventType string

const (
	// Added indicates a new resource was created.
	Added EventType = "ADDED"
	// Modified indicates an existing resource was updated.
	Modified EventType = "MODIFIED"
	// Deleted indicates a resource was removed.
	Deleted EventType = "DELETED"
)

// Event represents a change to a resource.
//
// Events flow through the system as:
//
//	HTTP POST /api/{resource}
//	      ↓
//	Storage.Create()
//	      ↓
//	EventBus.Publish(Event{Type: Added, ...})
//	      ↓
//	Watch clients (streaming)
//	Concurrent Controllers (reconciliation)
//
// This decouples API handlers from watchers and controllers.
type Event struct {
	// Type indicates what happened: Added, Modified, or Deleted.
	Type EventType `json:"type"`

	// Resource is the name of the resource that changed (e.g., "users", "orders").
	Resource string `json:"resource"`

	// Object is the resource object (after the change).
	// For Deleted events, this is the last state before deletion.
	Object any `json:"object"`

	// Timestamp is when the event was generated.
	Timestamp time.Time `json:"timestamp"`
}

// Subscription represents a client's subscription to events for a specific resource.
//
// Subscribers receive events through a channel and must actively drain the channel
// to avoid blocking other subscribers. The EventBus implementation ensures that
// no subscriber can block others.
type Subscription struct {
	// Resource is the resource name this subscription is for.
	Resource string

	// Events is the channel through which events are delivered.
	// The channel is buffered to handle brief processing delays.
	Events <-chan Event

	// done signals that the subscription should be closed.
	done chan struct{}

	// internal send channel (write-only) - closed by EventBus when unsubscribing.
	sendCh chan Event
}

// Close closes this subscription and stops receiving events.
// After closing, no more events will be sent.
func (s *Subscription) Close() error {
	select {
	case <-s.done:
		return nil
	default:
		close(s.done)
	}
	return nil
}
