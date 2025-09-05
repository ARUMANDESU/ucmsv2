package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

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

type StaffInvitationAcceptedAssertion struct {
	e *StaffInvitationAccepted
	t *testing.T
}

func NewStaffInvitationAcceptedAssertion(t *testing.T, e *StaffInvitationAccepted) *StaffInvitationAcceptedAssertion {
	return &StaffInvitationAcceptedAssertion{e: e, t: t}
}

func (a *StaffInvitationAcceptedAssertion) AssertStaffID(expected ID) *StaffInvitationAcceptedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.e.StaffID, "StaffID should match")
	return a
}

func (a *StaffInvitationAcceptedAssertion) AssertStaffBarcode(expected Barcode) *StaffInvitationAcceptedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.e.StaffBarcode, "StaffBarcode should match")
	return a
}

func (a *StaffInvitationAcceptedAssertion) AssertStaffUsername(expected string) *StaffInvitationAcceptedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.e.StaffUsername, "StaffUsername should match")
	return a
}

func (a *StaffInvitationAcceptedAssertion) AssertFirstName(expected string) *StaffInvitationAcceptedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.e.FirstName, "FirstName should match")
	return a
}

func (a *StaffInvitationAcceptedAssertion) AssertLastName(expected string) *StaffInvitationAcceptedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.e.LastName, "LastName should match")
	return a
}

func (a *StaffInvitationAcceptedAssertion) AssertEmail(expected string) *StaffInvitationAcceptedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.e.Email, "Email should match")
	return a
}

func (a *StaffInvitationAcceptedAssertion) AssertInvitationID(expected uuid.UUID) *StaffInvitationAcceptedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.e.InvitationID, "InvitationID should match")
	return a
}
