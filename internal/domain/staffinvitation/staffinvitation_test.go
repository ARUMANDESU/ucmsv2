package staffinvitation_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
)

// Test constants
const (
	testEmail1   = "testemail1@test.com"
	testEmail2   = "testemail2@test.com"
	invalidEmail = "invalid-email"
	validCode    = "valid-code"
	invalidCode  = "invalid-code"
)

// generateTestEmails creates a slice of test emails with the given count
func generateTestEmails(count int) []string {
	emails := make([]string, count)
	for i := range count {
		emails[i] = fmt.Sprintf("test%d@test.com", i+1)
	}
	return emails
}

func timePointer(t time.Time) *time.Time {
	return &t
}

// assertStaffInvitationFields validates all fields of a staff invitation
func assertStaffInvitationFields(t *testing.T, inv *staffinvitation.StaffInvitation, args staffinvitation.CreateArgs) {
	t.Helper()
	assert.NotEmpty(t, inv.ID())
	assert.NotEmpty(t, inv.Code())
	assert.Equal(t, args.RecipientsEmail, inv.RecipientsEmail())
	assert.Equal(t, args.CreatorID, inv.CreatorID())
	assert.Equal(t, args.ValidFrom, inv.ValidFrom())
	assert.Equal(t, args.ValidUntil, inv.ValidUntil())
	assert.NotZero(t, inv.CreatedAt())
	assert.Equal(t, inv.CreatedAt(), inv.UpdatedAt())
}

// assertCreatedEvent validates the Created event properties
func assertCreatedEvent(t *testing.T, inv *staffinvitation.StaffInvitation, event *staffinvitation.Created) {
	t.Helper()
	assert.Equal(t, inv.ID(), event.StaffInvitationID)
	assert.Equal(t, inv.Code(), event.Code)
	assert.Equal(t, inv.RecipientsEmail(), event.RecipientsEmail)
	assert.Equal(t, inv.CreatorID(), event.CreatorID)
	assert.Equal(t, inv.ValidFrom(), event.ValidFrom)
	assert.Equal(t, inv.ValidUntil(), event.ValidUntil)
}

func assertTimePointerWithinDuration(t *testing.T, expected, actual *time.Time, delta time.Duration) {
	t.Helper()
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		t.Fatalf("one of the time pointers is nil: expected=%v, actual=%v", expected, actual)
	}
	assert.WithinDuration(t, *expected, *actual, delta)
}

func TestNewStaffInvitation(t *testing.T) {
	t.Parallel()

	minuteLater := time.Now().Add(1 * time.Minute)
	twoMinutesLater := time.Now().Add(2 * time.Minute)
	minuteAgo := time.Now().Add(-1 * time.Minute)
	tests := []struct {
		name    string
		args    staffinvitation.CreateArgs
		wantErr error
	}{
		{
			name: "valid without validity time range",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
			},
		},
		{
			name: "valid with validFrom",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteLater,
			},
		},
		{
			name: "valid with validUntil",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
				ValidUntil:      &minuteLater,
			},
		},
		{
			name: "valid with validity time range",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteLater,
				ValidUntil:      &twoMinutesLater,
			},
		},
		{
			name: "valid with empty recipient emails",
			args: staffinvitation.CreateArgs{
				CreatorID: fixtures.TestStaff.ID,
			},
		},
		{
			name: "invalid with empty creator id",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
			},
			wantErr: validation.Errors{"creator_id": validation.ErrRequired},
		},
		{
			name: "invalid with invalid recipient email",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{invalidEmail, "valid@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validation.Errors{"0": is.ErrEmail} /* only the first invalid email is reported */},
		},
		{
			name: "invalid with duplicate recipient emails",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"duplicate@test.com", "duplicate@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validationx.ErrDuplicate},
		},
		{
			name: "invalid with validFrom in the past",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteAgo,
			},
			wantErr: validation.Errors{"valid_from": staffinvitation.ErrTimeInPast},
		},
		{
			name: "invalid with validUntil before validFrom",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &twoMinutesLater,
				ValidUntil:      &minuteLater,
			},
			wantErr: validation.Errors{"valid_until": staffinvitation.ErrTimeBeforeThreshold},
		},
		{
			name: "invalid with validUntil in the past",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
				ValidUntil:      &minuteAgo,
			},
			wantErr: validation.Errors{"valid_until": staffinvitation.ErrTimeInPast},
		},
		{
			name: "invalid with both validFrom and validUntil equal",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{testEmail1, testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteLater,
				ValidUntil:      &minuteLater,
			},
			wantErr: validation.Errors{"valid_until": staffinvitation.ErrTimeBeforeThreshold},
		},
		{
			name: "recipients email exceeds maximum",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: generateTestEmails(staffinvitation.MaxEmails + 1),
				CreatorID:       fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validation.ErrCountTooMany},
		},
		{
			name: "empty recipient email in the list",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"", testEmail2},
				CreatorID:       fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{
				"recipients_email": validation.Errors{"0": validation.ErrRequired},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			staffInvitation, err := staffinvitation.NewStaffInvitation(tt.args)
			if tt.wantErr != nil {
				require.Error(t, err)
				t.Logf("got err: %v", err)
				validationx.AssertValidationErrors(t, err, tt.wantErr)
				assert.Nil(t, staffInvitation)
			} else {
				require.NoError(t, err)
				require.NotNil(t, staffInvitation)

				fmt.Println(staffInvitation.Code())
				assertStaffInvitationFields(t, staffInvitation, tt.args)

				e := event.AssertSingleEvent[*staffinvitation.Created](t, staffInvitation.GetUncommittedEvents())
				assertCreatedEvent(t, staffInvitation, e)
			}
		})
	}
}

func TestStaffInvitation_UpdateRecipientsEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		staffInvitation   *staffinvitation.StaffInvitation
		userID            user.ID
		emails            []string
		wantErr           error
		isValidationErr   bool
		wantEmails        []string
		newEmails         []string
		isEventNotEmitted bool
	}{
		{
			name:            "valid update by the creator",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantEmails:      []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
		},
		{
			name:            "valid update with empty emails",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{},
			wantEmails:      []string{},
		},
		{
			name:            "invalid update by another staff",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff2.ID,
			emails:          []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantErr:         staffinvitation.ErrForbidden,
		},
		{
			name:            "invalid update with invalid recipient email",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{"invalid-email", fixtures.ValidStaff4Email},
			wantErr:         validation.Errors{"0": is.ErrEmail},
			isValidationErr: true,
		},
		{
			name:            "invalid update with duplicate recipient emails",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{fixtures.ValidStaff3Email, fixtures.ValidStaff3Email},
			wantErr:         validationx.ErrDuplicate,
			isValidationErr: true,
		},
		{
			name:            "recipients email exceeds maximum",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails: func() []string {
				emails := make([]string, 0, staffinvitation.MaxEmails+1)
				for range staffinvitation.MaxEmails + 1 {
					emails = append(emails, fmt.Sprintf("test%d@test.com", len(emails)+1))
				}
				return emails
			}(),
			wantErr:         validation.ErrCountTooMany,
			isValidationErr: true,
		},
		{
			name:            "empty recipient email in the list",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{"", fixtures.ValidStaff4Email},
			wantErr:         validation.Errors{"0": validation.ErrRequired},
			isValidationErr: true,
		},
		{
			name: "no change, thus no event is emitted",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.TestStaff2.Email}).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			userID:            fixtures.TestStaff.ID,
			emails:            []string{fixtures.TestStaff2.Email},
			wantEmails:        []string{fixtures.TestStaff2.Email},
			isEventNotEmitted: true,
		},
		{
			name:            "no change with empty emails",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{},
			wantEmails:      []string{},
		},
		{
			name:              "valid update to empty emails when already empty, thus no event is emitted",
			staffInvitation:   builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).WithRecipientsEmail([]string{}).Build(),
			userID:            fixtures.TestStaff.ID,
			emails:            []string{},
			wantEmails:        []string{},
			isEventNotEmitted: true,
		},
		{
			name:            "valid update from empty emails",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).WithRecipientsEmail([]string{}).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantEmails:      []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
		},
		{
			name: "valid update to the same emails",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				Build(),
			userID:            fixtures.TestStaff.ID,
			emails:            []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantEmails:        []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			isEventNotEmitted: true,
		},
		{
			name: "valid update only one new email",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email}).
				Build(),
			userID:     fixtures.TestStaff.ID,
			emails:     []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantEmails: []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			newEmails:  []string{fixtures.ValidStaff4Email},
		},
		{
			name: "valid update removing one email",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				Build(),
			userID:     fixtures.TestStaff.ID,
			emails:     []string{fixtures.ValidStaff3Email},
			wantEmails: []string{fixtures.ValidStaff3Email},
			newEmails:  []string{},
		},
		{
			name: "valid update but no change in emails, thus no event is emitted",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				Build(),
			userID:            fixtures.TestStaff.ID,
			emails:            []string{fixtures.ValidStaff4Email, fixtures.ValidStaff3Email},
			wantEmails:        []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			newEmails:         []string{},
			isEventNotEmitted: true,
		},
		{
			name: "invalid already deleted",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithDeletedAt(timePointer(time.Now().Add(-1 * time.Minute))).
				Build(),
			userID:  fixtures.TestStaff.ID,
			emails:  []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantErr: staffinvitation.ErrNotFoundOrDeleted,
		},
		{
			name: "invalid already deleted with non creator",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithDeletedAt(timePointer(time.Now().Add(-1 * time.Minute))).
				Build(),
			userID:  fixtures.TestStaff2.ID,
			emails:  []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantErr: staffinvitation.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.staffInvitation.UpdateRecipients(tt.userID, tt.emails)
			if tt.wantErr != nil {
				require.Error(t, err)
				if tt.isValidationErr {
					validationx.AssertValidationError(t, err, tt.wantErr)
				} else {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				assert.Equal(t, tt.staffInvitation.RecipientsEmail(), tt.staffInvitation.RecipientsEmail()) // no change
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.wantEmails, tt.staffInvitation.RecipientsEmail())

				events := tt.staffInvitation.GetUncommittedEvents()
				if !tt.isEventNotEmitted {

					e := event.AssertSingleEvent[*staffinvitation.RecipientsUpdated](t, events)
					assert.Equal(t, tt.staffInvitation.ID(), e.StaffInvitationID)
					assert.NotEmpty(t, e.Code)
					assert.Equal(t, tt.wantEmails, e.CurrentRecipientsEmail)
					if tt.newEmails != nil {
						assert.ElementsMatch(t, tt.newEmails, e.NewRecipientsEmail)
					} else {
						assert.ElementsMatch(t, tt.emails, e.NewRecipientsEmail)
					}
				} else {
					event.AssertNoEvents(t, events)
				}
			}
		})
	}
}

func TestStaffInvitation_UpdateValidity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		staffInvitation   *staffinvitation.StaffInvitation
		userID            user.ID
		validFrom         *time.Time
		validUntil        *time.Time
		wantErr           error
		isValidationErr   bool
		wantValidFrom     *time.Time
		wantValidUntil    *time.Time
		isEventNotEmitted bool
	}{
		{
			name:            "valid update by the creator to set both validFrom and validUntil",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			validFrom:       timePointer(time.Now().Add(1 * time.Minute)),
			validUntil:      timePointer(time.Now().Add(2 * time.Minute)),
			wantValidFrom:   timePointer(time.Now().Add(1 * time.Minute)),
			wantValidUntil:  timePointer(time.Now().Add(2 * time.Minute)),
		},
		{
			name:            "valid update by the creator to set only validFrom",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			validFrom:       timePointer(time.Now().Add(1 * time.Minute)),
			validUntil:      nil,
			wantValidFrom:   timePointer(time.Now().Add(1 * time.Minute)),
			wantValidUntil:  nil,
		},
		{
			name:            "valid update by the creator to set only validUntil",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			validFrom:       nil,
			validUntil:      timePointer(time.Now().Add(2 * time.Minute)),
			wantValidFrom:   nil,
			wantValidUntil:  timePointer(time.Now().Add(2 * time.Minute)),
		},
		{
			name: "valid update by the creator to clear both validFrom and validUntil",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithValidFrom(timePointer(time.Now().Add(1 * time.Minute))).Build(),
			userID:         fixtures.TestStaff.ID,
			validFrom:      nil,
			validUntil:     nil,
			wantValidFrom:  nil,
			wantValidUntil: nil,
		},
		{
			name:            "invalid update by another staff",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff2.ID,
			validFrom:       timePointer(time.Now().Add(1 * time.Minute)),
			validUntil:      timePointer(time.Now().Add(2 * time.Minute)),
			wantErr:         staffinvitation.ErrForbidden,
			wantValidFrom:   nil,
			wantValidUntil:  nil,
		},
		{
			name:            "invalid update with validFrom in the past",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			validFrom:       timePointer(time.Now().Add(-1 * time.Minute)),
			validUntil:      timePointer(time.Now().Add(1 * time.Minute)),
			wantErr:         staffinvitation.ErrTimeInPast,
			isValidationErr: true,
			wantValidFrom:   nil,
			wantValidUntil:  nil,
		},
		{
			name:            "invalid update with validUntil in the past",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			validFrom:       timePointer(time.Now().Add(1 * time.Minute)),
			validUntil:      timePointer(time.Now().Add(-1 * time.Minute)),
			wantErr:         staffinvitation.ErrTimeInPast,
			isValidationErr: true,
			wantValidFrom:   nil,
			wantValidUntil:  nil,
		},
		{
			name:            "invalid update with validUntil before validFrom",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			validFrom:       timePointer(time.Now().Add(2 * time.Minute)),
			validUntil:      timePointer(time.Now().Add(1 * time.Minute)),
			wantErr:         staffinvitation.ErrTimeBeforeThreshold,
			isValidationErr: true,
			wantValidFrom:   nil,
			wantValidUntil:  nil,
		},
		{
			name: "no change, thus no event is emitted",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithValidFrom(timePointer(time.Now().Add(1 * time.Minute))).Build(),
			userID:            fixtures.TestStaff.ID,
			validFrom:         timePointer(time.Now().Add(1 * time.Minute)),
			validUntil:        nil,
			wantValidFrom:     timePointer(time.Now().Add(1 * time.Minute)),
			wantValidUntil:    nil,
			isEventNotEmitted: true,
		},
		{
			name: "invalid already deleted",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithDeletedAt(timePointer(time.Now().Add(-1 * time.Minute))).
				Build(),
			userID:         fixtures.TestStaff.ID,
			validFrom:      timePointer(time.Now().Add(1 * time.Minute)),
			validUntil:     timePointer(time.Now().Add(2 * time.Minute)),
			wantErr:        staffinvitation.ErrNotFoundOrDeleted,
			wantValidFrom:  nil,
			wantValidUntil: nil,
		},
		{
			name: "invalid already deleted with non creator",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithDeletedAt(timePointer(time.Now().Add(-1 * time.Minute))).
				Build(),
			userID:         fixtures.TestStaff2.ID,
			validFrom:      timePointer(time.Now().Add(1 * time.Minute)),
			validUntil:     timePointer(time.Now().Add(2 * time.Minute)),
			wantErr:        staffinvitation.ErrForbidden,
			wantValidFrom:  nil,
			wantValidUntil: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.staffInvitation.UpdateValidity(tt.userID, tt.validFrom, tt.validUntil)
			if tt.wantErr != nil {
				require.Error(t, err)
				if tt.isValidationErr {
					validationx.AssertValidationError(t, err, tt.wantErr)
				} else {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				assert.Equal(t, tt.staffInvitation.ValidFrom(), tt.staffInvitation.ValidFrom())   // no change
				assert.Equal(t, tt.staffInvitation.ValidUntil(), tt.staffInvitation.ValidUntil()) // no change
			} else {
				require.NoError(t, err)
				assertTimePointerWithinDuration(t, tt.wantValidFrom, tt.staffInvitation.ValidFrom(), time.Second)
				assertTimePointerWithinDuration(t, tt.wantValidUntil, tt.staffInvitation.ValidUntil(), time.Second)

				events := tt.staffInvitation.GetUncommittedEvents()
				if !tt.isEventNotEmitted {
					e := event.AssertSingleEvent[*staffinvitation.ValidityUpdated](t, events)
					assert.Equal(t, tt.staffInvitation.ID(), e.StaffInvitationID)
					assertTimePointerWithinDuration(t, tt.wantValidFrom, e.ValidFrom, time.Second)
					assertTimePointerWithinDuration(t, tt.wantValidUntil, e.ValidUntil, time.Second)
				} else {
					event.AssertNoEvents(t, events)
				}
			}
		})
	}
}

func TestStaffInvitation_MarkDeleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		staffInvitation   *staffinvitation.StaffInvitation
		userID            user.ID
		wantErr           error
		isEventNotEmitted bool
	}{
		{
			name:            "valid delete by the creator",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
		},
		{
			name:            "invalid delete by another staff",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff2.ID,
			wantErr:         staffinvitation.ErrForbidden,
		},
		{
			name: "invalid delete when already deleted",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithCreatorID(fixtures.TestStaff.ID).
				WithDeletedAt(timePointer(time.Now().Add(-1 * time.Minute))).
				Build(),
			userID:            fixtures.TestStaff.ID,
			wantErr:           nil, // idempotent
			isEventNotEmitted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.staffInvitation.MarkDeleted(tt.userID)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, tt.staffInvitation.DeletedAt()) // not deleted
			} else {
				require.NoError(t, err)
				require.NotNil(t, tt.staffInvitation.DeletedAt())

				events := tt.staffInvitation.GetUncommittedEvents()
				if !tt.isEventNotEmitted {
					e := event.AssertSingleEvent[*staffinvitation.Deleted](t, events)
					assert.Equal(t, tt.staffInvitation.ID(), e.StaffInvitationID)
				} else {
					event.AssertNoEvents(t, events)
				}
			}
		})
	}
}

func TestStaffInvitation_ValidateInvitationAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		staffInvitation *staffinvitation.StaffInvitation
		email           string
		code            string
		wantErr         error
	}{
		{
			name: "valid access",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			email:   fixtures.ValidStaff3Email,
			code:    validCode,
			wantErr: nil,
		},
		{
			name: "invalid access with wrong code",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			email:   fixtures.ValidStaff3Email,
			code:    invalidCode,
			wantErr: staffinvitation.ErrInvalidInvitation,
		},
		{
			name: "invalid access with empty code",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			email:   fixtures.ValidStaff3Email,
			code:    "",
			wantErr: staffinvitation.ErrInvalidInvitation,
		},
		{
			name: "invalid access with empty email",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			email:   "",
			code:    validCode,
			wantErr: staffinvitation.ErrInvalidInvitation,
		},
		{
			name: "invalid access with empty email and code",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			email:   "",
			code:    "",
			wantErr: staffinvitation.ErrInvalidInvitation,
		},
		{
			name: "invalid access with email not in recipients",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.ValidStaff4Email}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			email:   fixtures.ValidStaff3Email,
			code:    validCode,
			wantErr: staffinvitation.ErrInvalidInvitation,
		},
		{
			name: "invalid access when already deleted",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				WithDeletedAt(timePointer(time.Now().Add(-1 * time.Minute))).
				Build(),
			email:   fixtures.ValidStaff3Email,
			code:    validCode,
			wantErr: staffinvitation.ErrNotFoundOrDeleted,
		},
		{
			name: "invalid access with empty recipient emails",
			staffInvitation: builders.NewStaffInvitationBuilder().
				WithRecipientsEmail([]string{}).
				WithCode(validCode).
				WithCreatorID(fixtures.TestStaff.ID).
				Build(),
			email:   fixtures.ValidStaff3Email,
			code:    validCode,
			wantErr: staffinvitation.ErrInvalidInvitation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.staffInvitation.ValidateInvitationAccess(tt.email, tt.code)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
