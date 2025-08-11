package postgres

import (
	"time"

	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type UserDTO struct {
	ID        string
	RoleID    int
	FirstName string
	LastName  string
	Email     string
	AvatarURL string
	Passhash  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
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

func DomainToUserDTO(u *user.User, roleID int) UserDTO {
	return UserDTO{
		ID:        string(u.ID()),
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
