package postgres

import (
	"time"

	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type UserDTO struct {
	ID        uuid.UUID
	Barcode   string
	Username  string
	RoleID    int
	FirstName string
	LastName  string
	Email     string
	AvatarURL string
	Passhash  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StudentDTO struct {
	ID      uuid.UUID
	GroupID uuid.UUID
}

type StaffDTO struct {
	ID uuid.UUID
}

type GlobalRoleDTO struct {
	ID   int16
	Name string
}

type RegistrationDTO struct {
	ID               uuid.UUID
	Email            string
	Status           string
	VerificationCode string
	CodeAttempts     int16
	CodeExpiresAt    time.Time
	ResendTimeout    time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type GroupDTO struct {
	ID        uuid.UUID
	Name      string
	Major     string
	Year      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StaffInvitationDTO struct {
	ID              uuid.UUID
	CreatorID       uuid.UUID
	Code            string
	RecipientsEmail []string
	ValidFrom       *time.Time
	ValidUntil      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

func DomainToRegistrationDTO(r *registration.Registration) RegistrationDTO {
	return RegistrationDTO{
		ID:               uuid.UUID(r.ID()),
		Email:            r.Email(),
		Status:           string(r.Status()),
		VerificationCode: r.VerificationCode(),
		CodeAttempts:     int16(r.CodeAttempts()),
		CodeExpiresAt:    r.CodeExpiresAt(),
		ResendTimeout:    r.ResendTimeout(),
		CreatedAt:        r.CreatedAt(),
		UpdatedAt:        r.UpdatedAt(),
	}
}

func RegistrationToDomain(dto RegistrationDTO) *registration.Registration {
	return registration.Rehydrate(registration.RehydrateArgs{
		ID:               registration.ID(dto.ID),
		Email:            dto.Email,
		Status:           registration.Status(dto.Status),
		VerificationCode: dto.VerificationCode,
		CodeAttempts:     int8(dto.CodeAttempts),
		CodeExpiresAt:    dto.CodeExpiresAt,
		ResendTimeout:    dto.ResendTimeout,
		CreatedAt:        dto.CreatedAt,
		UpdatedAt:        dto.UpdatedAt,
	})
}

func DomainToUserDTO(u *user.User, roleID int) UserDTO {
	return UserDTO{
		ID:        uuid.UUID(u.ID()),
		Barcode:   string(u.Barcode()),
		Username:  u.Username(),
		RoleID:    roleID,
		FirstName: u.FirstName(),
		LastName:  u.LastName(),
		Email:     u.Email(),
		AvatarURL: u.AvatarUrl(),
		Passhash:  u.PassHash(),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}

func UserToDomain(dto UserDTO, roleDTO GlobalRoleDTO) *user.User {
	return user.RehydrateUser(user.RehydrateUserArgs{
		ID:        user.ID(dto.ID),
		Barcode:   user.Barcode(dto.Barcode),
		Username:  dto.Username,
		FirstName: dto.FirstName,
		LastName:  dto.LastName,
		Role:      role.Global(roleDTO.Name),
		AvatarURL: dto.AvatarURL,
		Email:     dto.Email,
		PassHash:  dto.Passhash,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	})
}

func GroupToDomain(dto GroupDTO) *group.Group {
	return group.Rehydrate(group.RehydrateArgs{
		ID:    group.ID(dto.ID),
		Name:  dto.Name,
		Major: major.Major(dto.Major),
		Year:  dto.Year,
	})
}

func DomainToStaffInvitationDTO(i *staffinvitation.StaffInvitation) StaffInvitationDTO {
	return StaffInvitationDTO{
		ID:              uuid.UUID(i.ID()),
		CreatorID:       uuid.UUID(i.CreatorID()),
		Code:            i.Code(),
		RecipientsEmail: i.RecipientsEmail(),
		ValidFrom:       i.ValidFrom(),
		ValidUntil:      i.ValidUntil(),
		CreatedAt:       i.CreatedAt(),
		UpdatedAt:       i.UpdatedAt(),
		DeletedAt:       i.DeletedAt(),
	}
}

func StaffInvitationToDomain(dto StaffInvitationDTO) *staffinvitation.StaffInvitation {
	return staffinvitation.Rehydrate(staffinvitation.RehydrateArgs{
		ID:              staffinvitation.ID(dto.ID),
		CreatorID:       user.ID(dto.CreatorID),
		Code:            dto.Code,
		RecipientsEmail: dto.RecipientsEmail,
		ValidFrom:       dto.ValidFrom,
		ValidUntil:      dto.ValidUntil,
		CreatedAt:       dto.CreatedAt,
		UpdatedAt:       dto.UpdatedAt,
		DeletedAt:       dto.DeletedAt,
	})
}
