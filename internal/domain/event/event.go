package event

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Event interface {
	GetEventHeader() Header
	GetStreamName() string
}

type Header struct {
	ID        uuid.UUID
	Timestamp time.Time
	Metadata  map[string]string
}

func (e *Header) GetEventHeader() Header {
	return *e
}

func NewEventHeader() Header {
	return Header{
		ID:        uuid.New(),
		Timestamp: time.Now(),
	}
}

type Recorder struct {
	events []Event
}

func (e *Recorder) AddEvent(event Event) {
	if e == nil {
		return
	}
	e.events = append(e.events, event)
}

func (e *Recorder) GetUncommittedEvents() []Event {
	if e == nil {
		return nil
	}
	return e.events
}

func (e *Recorder) MarkEventsAsCommitted() {
	if e == nil {
		return
	}
	e.events = []Event{}
}

// AssertSingleEvent checks that exactly one event of the expected type was emitted
func AssertSingleEvent[T Event](t *testing.T, events []Event) T {
	t.Helper()
	require.Len(t, events, 1)
	event, ok := events[0].(T)
	require.True(t, ok, "expected event type %T, got %T", new(T), events[0])
	return event
}

// AssertNoEvents checks that no events were emitted
func AssertNoEvents(t *testing.T, events []Event) {
	t.Helper()
	assert.Empty(t, events, "expected no events to be emitted")
}
