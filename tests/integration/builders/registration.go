package builders

import (
	"time"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/pkg/randcode"
)

type RegistrationBuilder struct {
	id               registration.ID
	email            string
	status           registration.Status
	verificationCode string
	codeAttempts     int8
	codeExpiresAt    time.Time
	resendTimeout    time.Time
	createdAt        time.Time
	updatedAt        time.Time
}

func NewRegistrationBuilder() *RegistrationBuilder {
	code, _ := randcode.GenerateAlphaNumericCode(6)
	now := time.Now()

	return &RegistrationBuilder{
		id:               registration.NewID(),
		email:            "test@example.com",
		status:           registration.StatusPending,
		verificationCode: code,
		codeAttempts:     0,
		codeExpiresAt:    now.Add(10 * time.Minute),
		resendTimeout:    now.Add(1 * time.Minute),
		createdAt:        now,
		updatedAt:        now,
	}
}

func (b *RegistrationBuilder) WithID(id registration.ID) *RegistrationBuilder {
	b.id = id
	return b
}

func (b *RegistrationBuilder) WithEmail(email string) *RegistrationBuilder {
	b.email = email
	return b
}

func (b *RegistrationBuilder) WithStatus(status registration.Status) *RegistrationBuilder {
	b.status = status
	return b
}

func (b *RegistrationBuilder) WithVerificationCode(code string) *RegistrationBuilder {
	b.verificationCode = code
	return b
}

func (b *RegistrationBuilder) WithCodeAttempts(attempts int8) *RegistrationBuilder {
	b.codeAttempts = attempts
	return b
}

func (b *RegistrationBuilder) WithExpiredCode() *RegistrationBuilder {
	b.codeExpiresAt = time.Now().Add(-1 * time.Hour)
	return b
}

func (b *RegistrationBuilder) WithResendAvailable() *RegistrationBuilder {
	b.resendTimeout = time.Now().Add(-1 * time.Minute)
	return b
}

func (b *RegistrationBuilder) Completed() *RegistrationBuilder {
	b.status = registration.StatusCompleted
	return b
}

func (b *RegistrationBuilder) Expired() *RegistrationBuilder {
	b.status = registration.StatusExpired
	b.codeExpiresAt = time.Now().Add(-1 * time.Hour)
	return b
}

func (b *RegistrationBuilder) Build() *registration.Registration {
	return registration.Rehydrate(registration.RehydrateArgs{
		ID:               b.id,
		Email:            b.email,
		Status:           b.status,
		VerificationCode: b.verificationCode,
		CodeAttempts:     b.codeAttempts,
		CodeExpiresAt:    b.codeExpiresAt,
		ResendTimeout:    b.resendTimeout,
		CreatedAt:        b.createdAt,
		UpdatedAt:        b.updatedAt,
	})
}

func (b *RegistrationBuilder) BuildNew() (*registration.Registration, error) {
	return registration.NewRegistration(b.email, env.Test)
}

// Factory for common registration scenarios
type RegistrationFactory struct{}

func (f *RegistrationFactory) PendingRegistration(email string) *registration.Registration {
	return NewRegistrationBuilder().
		WithEmail(email).
		Build()
}

func (f *RegistrationFactory) ExpiredRegistration(email string) *registration.Registration {
	return NewRegistrationBuilder().
		WithEmail(email).
		Expired().
		Build()
}

func (f *RegistrationFactory) CompletedRegistration(email string) *registration.Registration {
	return NewRegistrationBuilder().
		WithEmail(email).
		Completed().
		Build()
}

func (f *RegistrationFactory) RegistrationWithFailedAttempts(email string, attempts int8) *registration.Registration {
	return NewRegistrationBuilder().
		WithEmail(email).
		WithCodeAttempts(attempts).
		Build()
}
