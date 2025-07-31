package email

import "github.com/ARUMANDESU/ucms/internal/domain/event"

const EventStreamName = "events_email"

type VerificationCodeRequested struct {
	event.Header
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (e VerificationCodeRequested) GetStreamName() string {
	return EventStreamName
}
