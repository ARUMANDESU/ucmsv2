package builders

import (
	"time"

	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/randcode"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
)

type StaffInvitationBuilder struct {
	id              staffinvitation.ID
	code            string
	recipientsEmail []string
	validFrom       *time.Time
	validUntil      *time.Time
	creatorID       user.ID
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

func NewStaffInvitationBuilder() *StaffInvitationBuilder {
	code, _ := randcode.GenerateAlphaNumericCode(staffinvitation.CodeLength)
	return &StaffInvitationBuilder{
		id:              staffinvitation.NewID(),
		code:            code,
		recipientsEmail: []string{fixtures.TestStaff2.Email},
		creatorID:       fixtures.TestStaff.ID,
		createdAt:       time.Now(),
		updatedAt:       time.Now(),
	}
}

func (b *StaffInvitationBuilder) WithID(id staffinvitation.ID) *StaffInvitationBuilder {
	b.id = id
	return b
}

func (b *StaffInvitationBuilder) WithCode(code string) *StaffInvitationBuilder {
	b.code = code
	return b
}

func (b *StaffInvitationBuilder) WithRecipientsEmail(recipientsEmail []string) *StaffInvitationBuilder {
	b.recipientsEmail = recipientsEmail
	return b
}

func (b *StaffInvitationBuilder) WithValidFrom(validFrom *time.Time) *StaffInvitationBuilder {
	b.validFrom = validFrom
	return b
}

func (b *StaffInvitationBuilder) WithValidUntil(validUntil *time.Time) *StaffInvitationBuilder {
	b.validUntil = validUntil
	return b
}

func (b *StaffInvitationBuilder) WithCreatorID(creatorID user.ID) *StaffInvitationBuilder {
	b.creatorID = creatorID
	return b
}

func (b *StaffInvitationBuilder) WithCreatedAt(createdAt time.Time) *StaffInvitationBuilder {
	b.createdAt = createdAt
	return b
}

func (b *StaffInvitationBuilder) WithUpdatedAt(updatedAt time.Time) *StaffInvitationBuilder {
	b.updatedAt = updatedAt
	return b
}

func (b *StaffInvitationBuilder) WithDeletedAt(deletedAt *time.Time) *StaffInvitationBuilder {
	b.deletedAt = deletedAt
	return b
}

func (b *StaffInvitationBuilder) Build() *staffinvitation.StaffInvitation {
	return staffinvitation.Rehydrate(staffinvitation.RehydrateArgs{
		ID:              b.id,
		Code:            b.code,
		RecipientsEmail: b.recipientsEmail,
		ValidFrom:       b.validFrom,
		ValidUntil:      b.validUntil,
		CreatorID:       b.creatorID,
		CreatedAt:       b.createdAt,
		UpdatedAt:       b.updatedAt,
		DeletedAt:       b.deletedAt,
	})
}
