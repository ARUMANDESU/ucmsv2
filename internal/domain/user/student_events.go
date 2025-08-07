package user

import (
	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
)

const (
	StudentEventStreamName = "events_student"
)

type StudentRegistered struct {
	event.Header
	StudentID ID
	Email     string
	FirstName string
	LastName  string
	GroupID   uuid.UUID
}

func (e *StudentRegistered) GetStreamName() string {
	return StudentEventStreamName
}
