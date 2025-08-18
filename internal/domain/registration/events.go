package registration

import (
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

func (e *RegistrationStarted) GetStreamName() string {
	return EventStreamName
}

type EmailVerified struct {
	event.Header
	event.Otel
	RegistrationID ID     `json:"registration_id"`
	Email          string `json:"email"`
}

func (e *EmailVerified) GetStreamName() string {
	return EventStreamName
}

type RegistrationFailed struct {
	event.Header
	event.Otel
	RegistrationID ID     `json:"registration_id"`
	Reason         string `json:"reason"`
}

func (e *RegistrationFailed) GetStreamName() string {
	return EventStreamName
}

type VerificationCodeResent struct {
	event.Header
	event.Otel
	RegistrationID   ID     `json:"registration_id"`
	Email            string `json:"email"`
	VerificationCode string `json:"verification_code"`
}

func (e *VerificationCodeResent) GetStreamName() string {
	return EventStreamName
}
