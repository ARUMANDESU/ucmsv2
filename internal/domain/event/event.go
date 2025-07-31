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
