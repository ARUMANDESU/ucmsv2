package staffinvitation_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
)

func TestNewStaffInvitation(t *testing.T) {
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
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
			},
		},
		{
			name: "valid with validFrom",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteLater,
			},
		},
		{
			name: "valid with validUntil",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidUntil:      &minuteLater,
			},
		},
		{
			name: "valid with validity time range",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
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
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
			},
			wantErr: validation.Errors{"creator_id": validation.ErrRequired},
		},
		{
			name: "invalid with invalid recipient email",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"invalid-email", "valid@test.com"},
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
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteAgo,
			},
			wantErr: validation.Errors{"valid_from": staffinvitation.ErrTimeInPast},
		},
		{
			name: "invalid with validUntil before validFrom",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &twoMinutesLater,
				ValidUntil:      &minuteLater,
			},
			wantErr: validation.Errors{"valid_until": staffinvitation.ErrTimeBeforeStart},
		},
		{
			name: "invalid with validUntil in the past",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidUntil:      &minuteAgo,
			},
			wantErr: validation.Errors{"valid_until": staffinvitation.ErrTimeInPast},
		},
		{
			name: "recipients email exceeds maximum",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: func() []string {
					emails := make([]string, 0, staffinvitation.MaxEmails+1)
					for range staffinvitation.MaxEmails + 1 {
						emails = append(emails, fmt.Sprintf("test%d@test.com", len(emails)+1))
					}
					return emails
				}(),
				CreatorID: fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validation.ErrCountTooMany},
		},
		{
			name: "empty recipient email in the list",
			args: staffinvitation.CreateArgs{
				RecipientsEmail: []string{"", "testemail2@test.com"},
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

				assert.NotEmpty(t, staffInvitation.ID())
				assert.NotEmpty(t, staffInvitation.Code())
				assert.Equal(t, tt.args.RecipientsEmail, staffInvitation.RecipientsEmail())
				assert.Equal(t, tt.args.CreatorID, staffInvitation.CreatorID())
				assert.Equal(t, tt.args.ValidFrom, staffInvitation.ValidFrom())
				assert.Equal(t, tt.args.ValidUntil, staffInvitation.ValidUntil())
				assert.NotZero(t, staffInvitation.CreatedAt())
				assert.Equal(t, staffInvitation.CreatedAt(), staffInvitation.UpdatedAt())

				events := staffInvitation.GetUncommittedEvents()
				require.Len(t, events, 1)
				e, ok := events[0].(*staffinvitation.Created)
				require.True(t, ok)
				assert.Equal(t, staffInvitation.ID(), e.StaffInvitationID)
				assert.Equal(t, staffInvitation.Code(), e.Code)
				assert.Equal(t, staffInvitation.RecipientsEmail(), e.RecipientsEmail)
				assert.Equal(t, staffInvitation.CreatorID(), e.CreatorID)
				assert.Equal(t, staffInvitation.ValidFrom(), e.ValidFrom)
				assert.Equal(t, staffInvitation.ValidUntil(), e.ValidUntil)
			}
		})
	}
}

func TestStaffInvitation_UpdateRecipientsEmail(t *testing.T) {
	tests := []struct {
		name            string
		staffInvitation *staffinvitation.StaffInvitation
		userID          user.ID
		emails          []string
		wantErr         error
		isValidationErr bool
		wantEmails      []string
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
			wantErr:         staffinvitation.ErrAccessDenied,
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
			name:            "no change",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{fixtures.TestStaff2.Email},
			wantEmails:      []string{fixtures.TestStaff2.Email},
		},
		{
			name:            "no change with empty emails",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{},
			wantEmails:      []string{},
		},
		{
			name:            "valid update to empty emails when already empty",
			staffInvitation: builders.NewStaffInvitationBuilder().WithCreatorID(fixtures.TestStaff.ID).WithRecipientsEmail([]string{}).Build(),
			userID:          fixtures.TestStaff.ID,
			emails:          []string{},
			wantEmails:      []string{},
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
			userID:     fixtures.TestStaff.ID,
			emails:     []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
			wantEmails: []string{fixtures.ValidStaff3Email, fixtures.ValidStaff4Email},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.staffInvitation.UpdateRecipients(tt.userID, tt.emails)
			if tt.wantErr != nil {
				require.Error(t, err)
				if tt.isValidationErr {
					// Unwrap the "validation failed:" wrapper to get the actual validation error
					unwrappedErr := err
					if wrappedErr, ok := err.(interface{ Unwrap() error }); ok {
						unwrappedErr = wrappedErr.Unwrap()
					}
					validationx.AssertValidationError(t, unwrappedErr, tt.wantErr)
				} else {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				assert.Equal(t, tt.staffInvitation.RecipientsEmail(), tt.staffInvitation.RecipientsEmail()) // no change
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantEmails, tt.staffInvitation.RecipientsEmail())

				events := tt.staffInvitation.GetUncommittedEvents()
				if len(events) > 0 {
					require.Len(t, events, 1)
					e, ok := events[0].(*staffinvitation.RecipientsUpdated)
					require.True(t, ok)
					assert.Equal(t, tt.staffInvitation.ID(), e.StaffInvitationID)
				}
			}
		})
	}
}
