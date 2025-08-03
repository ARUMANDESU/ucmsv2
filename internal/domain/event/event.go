package event

import (
	"time"

	"github.com/google/uuid"
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
