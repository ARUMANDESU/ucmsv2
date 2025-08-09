package mocks

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
)

type EventRepo struct {
	events   []event.Event
	eventsMu sync.Mutex
	eventCh  chan event.Event
}

func NewEventRepo() *EventRepo {
	return &EventRepo{
		events:  []event.Event{},
		eventCh: make(chan event.Event, 100),
	}
}

func (r *EventRepo) EventChannel() <-chan event.Event {
	return r.eventCh
}

func (r *EventRepo) Events() []event.Event {
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	eventsCopy := make([]event.Event, len(r.events))
	copy(eventsCopy, r.events)
	return eventsCopy
}

func (r *EventRepo) AssertEventNotExists(t *testing.T, e event.Event) *EventRepo {
	t.Helper()

	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	for _, ev := range r.events {
		if fmt.Sprintf("%T", ev) == fmt.Sprintf("%T", e) {
			t.Errorf("expected event %T to not exist, but it does", e)
			return r
		}
	}

	return r
}

func (r *EventRepo) AssertEventCount(t *testing.T, expectedCount int) *EventRepo {
	t.Helper()

	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	if len(r.events) != expectedCount {
		t.Errorf("expected %d events, but got %d", expectedCount, len(r.events))
	}

	return r
}

func (r *EventRepo) appendEvents(events ...event.Event) {
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	for _, e := range events {
		r.events = append(r.events, e)
		select {
		case r.eventCh <- e:
		default:
			// If the channel is full, we can choose to either drop the event or handle it
			// differently, depending on the use case. Here we just ignore it.
		}
	}
}

func RequireEventExists[T event.Event](t *testing.T, r *EventRepo, e T) T {
	t.Helper()

	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	tnil := *new(T)

	for _, ev := range r.events {
		if fmt.Sprintf("%T", ev) == fmt.Sprintf("%T", e) {
			if ev == nil {
				t.Errorf("expected event %T to not be nil, but it is", e)
				return tnil
			}
			header := ev.GetEventHeader()
			assert.NotEmpty(t, header, "event header should not be empty")
			return ev.(T)
		}
	}

	t.Fatalf("event %T not found in repository", e)

	return tnil
}
