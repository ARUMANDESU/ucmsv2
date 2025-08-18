package staffregtoken

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/randcode"
)

const (
	MaxEmails = 25
)

var (
	ErrTimeInPast         = errorx.NewInvalidRequest().WithKey("timestamp_in_past")
	ErrAtLeastOneEmail    = errorx.NewInvalidRequest().WithKey("at_least_one_email")
	ErrEmailAlreadyExists = errorx.NewDuplicateEntry().WithKey("email_already_exists_field")
	ErrMaxEmailsExceeded  = errorx.NewInvalidRequest().WithKey("max_emails_exceeded_field").WithArgs(map[string]any{"MaxEmails": MaxEmails})
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

type StaffRegToken struct {
	event.Recorder
	id        ID
	token     string
	emails    []string
	expiresAt *time.Time
	createdAt time.Time
	updatedAt time.Time
}

func NewStaffRegToken(emails []string, expiresAt *time.Time) (*StaffRegToken, error) {
	now := time.Now().UTC()

	if expiresAt != nil && expiresAt.Before(now) {
		return nil, ErrTimeInPast
	}
	if len(emails) == 0 {
		return nil, ErrAtLeastOneEmail
	}
	if len(emails) > MaxEmails {
		return nil, ErrMaxEmailsExceeded
	}

	token, err := randcode.GenerateAlphaNumericCode(10)
	if err != nil {
		return nil, err
	}

	return &StaffRegToken{
		id:        NewID(),
		token:     token,
		emails:    emails,
		expiresAt: expiresAt,
		createdAt: now,
		updatedAt: now,
	}, nil
}

type RehydrateArgs struct {
	ID        ID
	Token     string
	Emails    []string
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func Rehydrate(args RehydrateArgs) *StaffRegToken {
	return &StaffRegToken{
		id:        args.ID,
		token:     args.Token,
		emails:    args.Emails,
		expiresAt: args.ExpiresAt,
		createdAt: args.CreatedAt,
		updatedAt: args.UpdatedAt,
	}
}

func (s *StaffRegToken) AddEmails(emails []string) error {
	if len(emails) == 0 {
		return ErrAtLeastOneEmail
	}
	if len(s.emails) > MaxEmails {
		return ErrMaxEmailsExceeded
	}
	emap := s.emailsmap()
	for _, email := range emails {
		if _, ok := emap[email]; ok {
			return ErrEmailAlreadyExists.WithArgs(map[string]any{"Email": email})
		}
		// err := registration.ValidateEmail(email)
		// if err != nil {
		// 	return err
		// }
		emap[email] = struct{}{} // to avoid duplicates
	}

	s.emails = append(s.emails, emails...)
	s.updatedAt = time.Now().UTC()

	return nil
}

func (s *StaffRegToken) UpdateExpiresAt(expiresAt *time.Time) error {
	if expiresAt != nil && expiresAt.Before(time.Now().UTC()) {
		return ErrTimeInPast
	}

	s.expiresAt = expiresAt
	s.updatedAt = time.Now().UTC()

	return nil
}

func (s *StaffRegToken) emailsmap() map[string]struct{} {
	m := make(map[string]struct{}, len(s.emails))
	for _, emails := range s.emails {
		m[emails] = struct{}{}
	}
	return m
}
