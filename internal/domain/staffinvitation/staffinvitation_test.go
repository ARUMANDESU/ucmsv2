package staffinvitation

import (
	"testing"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/pkg/validationx"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
)

func TestNewStaffInvitation(t *testing.T) {
	minuteLater := time.Now().Add(1 * time.Minute)
	twoMinutesLater := time.Now().Add(2 * time.Minute)
	minuteAgo := time.Now().Add(-1 * time.Minute)
	tests := []struct {
		name    string
		args    CreateArgs
		wantErr error
	}{
		{
			name: "valid without validity time range",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
			},
		},
		{
			name: "valid with validFrom",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteLater,
			},
		},
		{
			name: "valid with validUntil",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidUntil:      &minuteLater,
			},
		},
		{
			name: "valid with validity time range",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteLater,
				ValidUntil:      &twoMinutesLater,
			},
		},
		{
			name: "invalid with empty creator id",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
			},
			wantErr: validation.Errors{"creator_id": validation.ErrRequired},
		},
		{
			name: "invalid with empty recipient emails",
			args: CreateArgs{
				CreatorID: fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validation.ErrRequired},
		},
		{
			name: "invalid with invalid recipient email",
			args: CreateArgs{
				RecipientsEmail: []string{"invalid-email", "valid@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validation.Errors{"0": is.ErrEmail} /* only the first invalid email is reported */},
		},
		{
			name: "invalid with duplicate recipient emails",
			args: CreateArgs{
				RecipientsEmail: []string{"duplicate@test.com", "duplicate@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validationx.ErrDuplicate},
		},
		{
			name: "invalid with validFrom in the past",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &minuteAgo,
			},
			wantErr: validation.Errors{"valid_from": ErrTimeInPast},
		},
		{
			name: "invalid with validUntil before validFrom",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidFrom:       &twoMinutesLater,
				ValidUntil:      &minuteLater,
			},
			wantErr: validation.Errors{"valid_until": ErrTimeBeforeStart},
		},
		{
			name: "invalid with validUntil in the past",
			args: CreateArgs{
				RecipientsEmail: []string{"testemail1@test.com", "testemail2@test.com"},
				CreatorID:       fixtures.TestStaff.ID,
				ValidUntil:      &minuteAgo,
			},
			wantErr: validation.Errors{"valid_until": ErrTimeInPast},
		},
		{
			name: "recipients email exceeds maximum",
			args: CreateArgs{
				RecipientsEmail: func() []string {
					emails := make([]string, 0, 101)
					for range MaxEmails + 1 {
						emails = append(emails, "testemail1@test.com")
					}
					return emails
				}(),
				CreatorID: fixtures.TestStaff.ID,
			},
			wantErr: validation.Errors{"recipients_email": validation.ErrLengthOutOfRange},
		},
		{
			name: "empty recipient email in the list",
			args: CreateArgs{
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
			staffInvitation, err := NewStaffInvitation(tt.args)
			if tt.wantErr != nil {
				require.Error(t, err)
				t.Logf("got error: %v", err)
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
			}
		})
	}
}
