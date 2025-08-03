package registration

import (
	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
)

const EventStreamName = "events_registration"

type RegistrationStarted struct {
	event.Header
	event.Otel
	RegistrationID   ID     `json:"registration_id"`
	Email            string `json:"email"`
	VerificationCode string `json:"verification_code"`
}

func (e RegistrationStarted) GetStreamName() string {
	return EventStreamName
}

type EmailVerified struct {
	event.Header
	event.Otel
	RegistrationID ID     `json:"registration_id"`
	Email          string `json:"email"`
}

func (e EmailVerified) GetStreamName() string {
	return EventStreamName
}

type StudentRegistrationCompleted struct {
	event.Header
	event.Otel
	RegistrationID ID        `json:"registration_id"`
	Barcode        string    `json:"barcode"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	PassHash       []byte    `json:"pass_hash"`
	GroupID        uuid.UUID `json:"group_id"`
}

func (e StudentRegistrationCompleted) GetStreamName() string {
	return EventStreamName
}

type StaffRegistrationCompleted struct {
	event.Header
	event.Otel
	RegistrationID ID     `json:"registration_id"`
	Barcode        string `json:"barcode"`
	Email          string `json:"email"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	PassHash       []byte `json:"pass_hash"`
}

func (e StaffRegistrationCompleted) GetStreamName() string {
	return EventStreamName
}

type RegistrationFailed struct {
	event.Header
	event.Otel
	RegistrationID ID     `json:"registration_id"`
	Reason         string `json:"reason"`
}

func (e RegistrationFailed) GetStreamName() string {
	return EventStreamName
}

type VerificationCodeResent struct {
	event.Header
	event.Otel
	RegistrationID   ID     `json:"registration_id"`
	Email            string `json:"email"`
	VerificationCode string `json:"verification_code"`
}

func (e VerificationCodeResent) GetStreamName() string {
	return EventStreamName
}
