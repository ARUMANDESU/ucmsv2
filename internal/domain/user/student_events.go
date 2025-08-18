package user

import (
	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
)

const (
	StudentEventStreamName = "events_student"
)

type StudentRegistered struct {
	event.Header
	event.Otel
	StudentID      ID
	RegistrationID registration.ID
	Email          string
	FirstName      string
	LastName       string
	GroupID        uuid.UUID
}

func (e *StudentRegistered) GetStreamName() string {
	return StudentEventStreamName
}
