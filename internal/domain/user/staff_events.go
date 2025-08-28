package user

import (
	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
)

const StaffEventStreamName = "events_staff"

type StaffRegistered struct {
	event.Header
	StaffID        ID
	StaffBarcode   Barcode
	RegistrationID registration.ID
	FirstName      string
	LastName       string
	Email          string
}

func (e *StaffRegistered) GetStreamName() string {
	return StaffEventStreamName
}
