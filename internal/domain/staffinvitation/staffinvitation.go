package staffinvitation

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/i18nx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/randcode"
	"gitlab.com/ucmsv2/ucms-backend/pkg/validationx"
)

const EventStreamName = "events_staff_invitation"

const (
	CodeLength         = 20
	MaxEmails          = 25
	ValidFromThreshold = time.Minute
)

var (
	ErrTimeInPast          = validation.NewError(i18nx.ValidationTimeInPast, i18nx.MsgValidationTimeInPastOther)
	ErrTimeBeforeThreshold = validation.NewError(i18nx.ValidationTimeBeforeThreshold, i18nx.MsgValidationTimeBeforeThresholdOther)
	ErrForbidden           = errorx.NewForbidden()
	ErrNotFoundOrDeleted   = errorx.NewNotFound().WithKey(i18nx.KeyNotFoundOrDeleted)
	ErrInvalidInvitation   = errorx.NewInvalidRequest().WithKey(i18nx.KeyInvalidInvitation)
)

var (
	recipientsEmailRules = []validation.Rule{
		validation.Count(0, MaxEmails),
		validationx.NoDuplicate,
		validation.Each(
			validation.Required,
			is.EmailFormat,
		),
	}
	validFromRules = func(validFrom *time.Time) []validation.Rule {
		rules := []validation.Rule{
			validation.NilOrNotEmpty,
		}
		if validFrom != nil {
			rules = append(rules, validation.Min(time.Now().UTC()).ErrorObject(ErrTimeInPast))
		}
		return rules
	}
	validUntilRules = func(validUntil *time.Time, validFrom *time.Time) []validation.Rule {
		rules := []validation.Rule{validation.NilOrNotEmpty}
		if validUntil != nil {
			rules = append(rules, validation.Min(time.Now().UTC()).ErrorObject(ErrTimeInPast))

			if validFrom != nil {
				rules = append(rules, validation.Min(validFrom.Add(ValidFromThreshold)).ErrorObject(ErrTimeBeforeThreshold))
			}
		}
		return rules
	}
)

type ID uuid.UUID

func NewID() ID {
	return ID(uuid.New())
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(id).String())
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	uid, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*id = ID(uid)
	return nil
}

type StaffInvitation struct {
	event.Recorder
	id              ID
	code            string
	recipientsEmail []string
	validFrom       *time.Time
	validUntil      *time.Time
	creatorID       user.ID
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

type CreateArgs struct {
	RecipientsEmail []string   `json:"recipients_email"`
	CreatorID       user.ID    `json:"creator_id"`
	ValidFrom       *time.Time `json:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until"`
}

func NewStaffInvitation(args CreateArgs) (*StaffInvitation, error) {
	const op = "staffinvitation.NewStaffInvitation"
	now := time.Now().UTC()

	err := validation.ValidateStruct(
		&args,
		validation.Field(&args.CreatorID, validationx.Required),
		validation.Field(&args.RecipientsEmail, recipientsEmailRules...),
		validation.Field(&args.ValidFrom, validFromRules(args.ValidFrom)...),
		validation.Field(&args.ValidUntil, validUntilRules(args.ValidUntil, args.ValidFrom)...),
	)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	token, err := randcode.GenerateAlphaNumericCode(CodeLength)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	staffInvitation := &StaffInvitation{
		id:              NewID(),
		code:            token,
		recipientsEmail: args.RecipientsEmail,
		validFrom:       args.ValidFrom,
		validUntil:      args.ValidUntil,
		creatorID:       args.CreatorID,
		createdAt:       now,
		updatedAt:       now,
	}

	staffInvitation.AddEvent(&Created{
		Header:            event.NewEventHeader(),
		StaffInvitationID: staffInvitation.id,
		Code:              staffInvitation.code,
		RecipientsEmail:   staffInvitation.recipientsEmail,
		ValidFrom:         staffInvitation.validFrom,
		ValidUntil:        staffInvitation.validUntil,
		CreatorID:         args.CreatorID,
	})

	return staffInvitation, nil
}

type RehydrateArgs struct {
	ID              ID
	Code            string
	RecipientsEmail []string
	ValidFrom       *time.Time
	ValidUntil      *time.Time
	CreatorID       user.ID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

func Rehydrate(args RehydrateArgs) *StaffInvitation {
	return &StaffInvitation{
		id:              args.ID,
		code:            args.Code,
		recipientsEmail: args.RecipientsEmail,
		validFrom:       args.ValidFrom,
		validUntil:      args.ValidUntil,
		creatorID:       args.CreatorID,
		createdAt:       args.CreatedAt,
		updatedAt:       args.UpdatedAt,
		deletedAt:       args.DeletedAt,
	}
}

func (s *StaffInvitation) UpdateRecipients(userID user.ID, emails []string) error {
	const op = "staffinvitation.StaffInvitation.UpdateRecipients"
	if s.creatorID != userID {
		return errorx.Wrap(ErrForbidden, op)
	}
	if s.deletedAt != nil {
		return errorx.Wrap(ErrNotFoundOrDeleted, op)
	}

	err := validation.Validate(emails, recipientsEmailRules...)
	if err != nil {
		return errorx.Wrap(err, op)
	}

	previousEmails := make(map[string]struct{}, len(s.recipientsEmail))
	for _, email := range s.recipientsEmail {
		previousEmails[email] = struct{}{}
	}

	if len(emails) == len(s.recipientsEmail) {
		same := true
		for _, email := range emails {
			if _, exists := previousEmails[email]; !exists {
				same = false
				break
			}
		}
		if same {
			return nil // No change needed
		}
	}

	newEmails := make([]string, 0, len(emails))
	for _, email := range emails {
		if _, exists := previousEmails[email]; !exists {
			newEmails = append(newEmails, email)
		}
	}

	s.recipientsEmail = emails
	s.updatedAt = time.Now().UTC()

	s.AddEvent(&RecipientsUpdated{
		Header:                 event.NewEventHeader(),
		StaffInvitationID:      s.id,
		Code:                   s.code,
		NewRecipientsEmail:     newEmails,
		CurrentRecipientsEmail: s.recipientsEmail,
	})

	return nil
}

func (s *StaffInvitation) UpdateValidity(userID user.ID, from *time.Time, until *time.Time) error {
	const op = "staffinvitation.StaffInvitation.UpdateValidity"
	if s.creatorID != userID {
		return errorx.Wrap(ErrForbidden, op)
	}
	if s.deletedAt != nil {
		return errorx.Wrap(ErrNotFoundOrDeleted, op)
	}

	if err := validation.Validate(from, validFromRules(from)...); err != nil {
		return errorx.Wrap(err, op)
	}
	if err := validation.Validate(until, validUntilRules(until, from)...); err != nil {
		return errorx.Wrap(err, op)
	}

	isValidFromSame := (s.validFrom == nil && from == nil) ||
		(s.validFrom != nil && from != nil && s.validFrom.Truncate(time.Second).Equal(from.Truncate(time.Second)))
	isValidUntilSame := (s.validUntil == nil && until == nil) ||
		(s.validUntil != nil && until != nil && s.validUntil.Truncate(time.Second).Equal(until.Truncate(time.Second)))
	if isValidFromSame && isValidUntilSame {
		return nil // No change needed
	}

	s.validFrom = from
	s.validUntil = until
	s.updatedAt = time.Now().UTC()

	s.AddEvent(&ValidityUpdated{
		Header:            event.NewEventHeader(),
		StaffInvitationID: s.id,
		ValidFrom:         s.validFrom,
		ValidUntil:        s.validUntil,
	})

	return nil
}

func (s *StaffInvitation) MarkDeleted(userID user.ID) error {
	const op = "staffinvitation.StaffInvitation.MarkDeleted"
	if s.creatorID != userID {
		return errorx.Wrap(ErrForbidden, op)
	}
	if s.deletedAt != nil {
		return nil
	}

	now := time.Now().UTC()
	s.deletedAt = &now

	s.AddEvent(&Deleted{
		Header:            event.NewEventHeader(),
		StaffInvitationID: s.id,
	})

	return nil
}

func (s *StaffInvitation) ValidateInvitationAccess(email, code string) error {
	const op = "staffinvitation.StaffInvitation.ValidateInvitationAccess"
	if s.deletedAt != nil {
		return errorx.Wrap(ErrNotFoundOrDeleted, op)
	}
	if email == "" || code == "" || s.code != code {
		return errorx.Wrap(ErrInvalidInvitation, op)
	}

	if slices.Contains(s.recipientsEmail, email) {
		return nil
	}

	return errorx.Wrap(ErrInvalidInvitation, op)
}

func (s *StaffInvitation) ID() ID {
	if s == nil {
		return ID{}
	}

	return s.id
}

func (s *StaffInvitation) Code() string {
	if s == nil {
		return ""
	}

	return s.code
}

func (s *StaffInvitation) RecipientsEmail() []string {
	if s == nil {
		return nil
	}

	return s.recipientsEmail
}

func (s *StaffInvitation) ValidFrom() *time.Time {
	if s == nil {
		return nil
	}

	return s.validFrom
}

func (s *StaffInvitation) ValidUntil() *time.Time {
	if s == nil {
		return nil
	}

	return s.validUntil
}

func (s *StaffInvitation) CreatorID() user.ID {
	if s == nil {
		return user.ID{}
	}

	return s.creatorID
}

func (s *StaffInvitation) CreatedAt() time.Time {
	if s == nil {
		return time.Time{}
	}

	return s.createdAt
}

func (s *StaffInvitation) UpdatedAt() time.Time {
	if s == nil {
		return time.Time{}
	}

	return s.updatedAt
}

func (s *StaffInvitation) DeletedAt() *time.Time {
	if s == nil {
		return nil
	}

	return s.deletedAt
}

type Created struct {
	event.Header
	event.Otel
	StaffInvitationID ID         `json:"staff_invitation_id"`
	Code              string     `json:"code"`
	RecipientsEmail   []string   `json:"recipients_email"`
	ValidFrom         *time.Time `json:"valid_from,omitempty"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
	CreatorID         user.ID    `json:"creator_id"`
}

func (e *Created) GetStreamName() string {
	return EventStreamName
}

type RecipientsUpdated struct {
	event.Header
	event.Otel
	StaffInvitationID      ID       `json:"staff_invitation_id"`
	Code                   string   `json:"code"`
	NewRecipientsEmail     []string `json:"new_recipients_email"`
	CurrentRecipientsEmail []string `json:"current_recipients_email"`
}

func (e *RecipientsUpdated) GetStreamName() string {
	return EventStreamName
}

type ValidityUpdated struct {
	event.Header
	event.Otel
	StaffInvitationID ID         `json:"staff_invitation_id"`
	ValidFrom         *time.Time `json:"valid_from,omitempty"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
}

func (e *ValidityUpdated) GetStreamName() string {
	return EventStreamName
}

type Deleted struct {
	event.Header
	event.Otel
	StaffInvitationID ID `json:"staff_invitation_id"`
}

func (e *Deleted) GetStreamName() string {
	return EventStreamName
}

type Assertion struct {
	t *testing.T
	s *StaffInvitation
}

func NewAssertion(t *testing.T, s *StaffInvitation) *Assertion {
	return &Assertion{t, s}
}

func (a *Assertion) AssertID(expected ID) *Assertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.s.id, "ID should match")
	return a
}

func (a *Assertion) AssertIDNotEmpty() *Assertion {
	a.t.Helper()
	assert.NotEqual(a.t, ID{}, a.s.id, "ID should not be empty")
	return a
}

func (a *Assertion) AssertCode(expected string) *Assertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.s.code, "Code should match")
	return a
}

func (a *Assertion) AssertCodeNotEmpty() *Assertion {
	a.t.Helper()
	assert.NotEmpty(a.t, a.s.code, "Code should not be empty")
	return a
}

func (a *Assertion) AssertRecipientsEmail(expected []string) *Assertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.s.recipientsEmail, "RecipientsEmail should match")
	return a
}

func (a *Assertion) AssertValidFrom(expected *time.Time) *Assertion {
	a.t.Helper()
	if expected == nil {
		assert.Nil(a.t, a.s.validFrom, "ValidFrom should be nil")
	} else {
		require.NotNil(a.t, a.s.validFrom, "ValidFrom should not be nil")
		assert.WithinDuration(a.t, *expected, *a.s.validFrom, time.Second, "ValidFrom should match")
	}
	return a
}

func (a *Assertion) AssertValidUntil(expected *time.Time) *Assertion {
	a.t.Helper()
	if expected == nil {
		assert.Nil(a.t, a.s.validUntil, "ValidUntil should be nil")
	} else {
		require.NotNil(a.t, a.s.validUntil, "ValidUntil should not be nil")
		assert.WithinDuration(a.t, *expected, *a.s.validUntil, time.Second, "ValidUntil should match")
	}
	return a
}

func (a *Assertion) AssertCreatorID(expected user.ID) *Assertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.s.creatorID, "CreatorID should match")
	return a
}

func (a *Assertion) AssertCreatedAt(expected time.Time) *Assertion {
	a.t.Helper()
	assert.WithinDuration(a.t, expected, a.s.createdAt, time.Second, "CreatedAt should match")
	return a
}

func (a *Assertion) AssertUpdatedAt(expected time.Time) *Assertion {
	a.t.Helper()
	assert.WithinDuration(a.t, expected, a.s.updatedAt, time.Second, "UpdatedAt should match")
	return a
}

func (a *Assertion) AssertDeletedAt(expected *time.Time) *Assertion {
	a.t.Helper()
	if expected == nil {
		assert.Nil(a.t, a.s.deletedAt, "DeletedAt should be nil")
	} else {
		assert.NotNil(a.t, a.s.deletedAt, "DeletedAt should not be nil")
		assert.WithinDuration(a.t, *expected, *a.s.deletedAt, time.Second, "DeletedAt should match")
	}
	return a
}

func (a *Assertion) AssertDeleted(expected bool) *Assertion {
	a.t.Helper()
	if expected {
		assert.NotNil(a.t, a.s.deletedAt, "StaffInvitation should be deleted")
	} else {
		assert.Nil(a.t, a.s.deletedAt, "StaffInvitation should not be deleted")
	}
	return a
}

func (a *Assertion) AssertEventCount(expected int) *Assertion {
	a.t.Helper()
	events := a.s.GetUncommittedEvents()
	assert.Len(a.t, events, expected, "Event count should match")
	return a
}
