package user

import (
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/group"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
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
