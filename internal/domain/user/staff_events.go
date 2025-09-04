package user

import (
	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
)

const StaffEventStreamName = "events_staff"

type StaffInvitationAccepted struct {
	event.Header
	event.Otel
	StaffID       ID
	StaffBarcode  Barcode
	StaffUsername string
	FirstName     string
	LastName      string
	Email         string
	InvitationID  uuid.UUID
}

func (e *StaffInvitationAccepted) GetStreamName() string {
	return StaffEventStreamName
}

type InitialStaffCreated struct {
	event.Header
	event.Otel
	StaffID       ID
	StaffBarcode  Barcode
	StaffUsername string
	FirstName     string
	LastName      string
	Email         string
}

func (e *InitialStaffCreated) GetStreamName() string {
	return StaffEventStreamName
}
