package user

import "github.com/ARUMANDESU/ucms/internal/domain/event"

const StaffEventStreamName = "events_staff"

type StaffRegistered struct {
	event.Header
	StaffID   ID
	FirstName string
	LastName  string
	Email     string
}

func (e *StaffRegistered) GetStreamName() string {
	return StaffEventStreamName
}
