package user

import (
	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
)

const (
	StudentEventStreamName = "events_student"
)

type StudentRegistered struct {
	event.Header
	event.Otel
	StudentID       ID
	StudentBarcode  Barcode
	StudentUsername string
	RegistrationID  registration.ID
	Email           string
	FirstName       string
	LastName        string
	GroupID         group.ID
}

func (e *StudentRegistered) GetStreamName() string {
	return StudentEventStreamName
}
