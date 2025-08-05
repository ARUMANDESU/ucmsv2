package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
)

type Helper struct {
	pool *pgxpool.Pool
}

func NewHelper(pool *pgxpool.Pool) *Helper {
	return &Helper{pool: pool}
}

func (h *Helper) TruncateAll(t *testing.T) {
	t.Helper()

	tables := []string{
		"registrations",
		"students",
		"users",
		"groups",
	}

	ctx := context.Background()
	for _, table := range tables {
		_, err := h.pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "failed to truncate table %s", table)
	}
}

func (h *Helper) AssertRegistrationExists(t *testing.T, email string) *RegistrationAssertion {
	t.Helper()

	var r RegistrationRow
	err := h.pool.QueryRow(context.Background(), `
        SELECT id, email, status, verification_code, code_attempts, 
               code_expires_at, resend_timeout, created_at, updated_at
        FROM registrations
        WHERE email = $1
    `, email).Scan(
		&r.ID, &r.Email, &r.Status, &r.VerificationCode,
		&r.CodeAttempts, &r.CodeExpiresAt, &r.ResendTimeout,
		&r.CreatedAt, &r.UpdatedAt,
	)

	require.NoError(t, err, "registration not found for email: %s", email)

	return &RegistrationAssertion{t: t, row: r, db: h}
}

func (h *Helper) AssertRegistrationNotExists(t *testing.T, email string) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM registrations WHERE email = $1", email).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "expected no registration for email %s, but found %d", email, count)
}

func (h *Helper) AssertRegistrationCount(t *testing.T, expected int) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM registrations").Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, expected, count, "unexpected registration count")
}

func (h *Helper) AssertUserExists(t *testing.T, email string) *UserAssertion {
	t.Helper()

	var u UserRow
	err := h.pool.QueryRow(context.Background(), `
        SELECT u.id, u.email, u.first_name, u.last_name, u.role_id, 
               u.avatar_url, u.pass_hash, u.created_at, u.updated_at,
               gr.name as role_name
        FROM users u
        JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.email = $1
    `, email).Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.RoleID,
		&u.AvatarURL, &u.PassHash, &u.CreatedAt, &u.UpdatedAt, &u.RoleName,
	)

	require.NoError(t, err, "user not found for email: %s", email)

	return &UserAssertion{t: t, row: u, db: h}
}

func (h *Helper) AssertUserNotExists(t *testing.T, email string) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "expected no user for email %s", email)
}

func (h *Helper) AssertStudentExists(t *testing.T, userID string) *StudentAssertion {
	t.Helper()

	var s StudentRow
	err := h.pool.QueryRow(context.Background(), `
        SELECT s.user_id, s.group_id, s.created_at, s.updated_at,
               g.name as group_name, g.year, g.major
        FROM students s
        JOIN groups g ON s.group_id = g.id
        WHERE s.user_id = $1
    `, userID).Scan(
		&s.UserID, &s.GroupID, &s.CreatedAt, &s.UpdatedAt,
		&s.GroupName, &s.Year, &s.Major,
	)

	require.NoError(t, err, "student not found for user_id: %s", userID)

	return &StudentAssertion{t: t, row: s}
}

func (h *Helper) QueryOne(t *testing.T, query string, args ...any) pgx.Row {
	t.Helper()
	return h.pool.QueryRow(context.Background(), query, args...)
}

func (h *Helper) Query(t *testing.T, query string, args ...any) (pgx.Rows, func()) {
	t.Helper()

	rows, err := h.pool.Query(context.Background(), query, args...)
	require.NoError(t, err)

	return rows, func() { rows.Close() }
}

func (h *Helper) Exec(t *testing.T, query string, args ...any) pgconn.CommandTag {
	t.Helper()

	tag, err := h.pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)

	return tag
}

func (h *Helper) SeedRegistration(t *testing.T, r *registration.Registration) {
	t.Helper()

	_, err := h.pool.Exec(context.Background(), `
        INSERT INTO registrations (id, email, status, verification_code, 
                                 code_attempts, code_expires_at, resend_timeout, 
                                 created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, uuid.UUID(r.ID()), r.Email(), string(r.Status()), r.VerificationCode(),
		r.CodeAttempts(), r.CodeExpiresAt(), r.ResendTimeout(),
		r.CreatedAt(), r.UpdatedAt())

	require.NoError(t, err)
}

func (h *Helper) SeedUser(t *testing.T, u *user.User) {
	t.Helper()

	var roleID int16
	err := h.pool.QueryRow(context.Background(),
		"SELECT id FROM global_roles WHERE name = $1", string(u.Role())).Scan(&roleID)
	require.NoError(t, err)

	_, err = h.pool.Exec(context.Background(), `
        INSERT INTO users (id, email, role_id, first_name, last_name, 
                          avatar_url, pass_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, string(u.ID()), u.Email(), roleID, u.FirstName(), u.LastName(),
		u.AvatarUrl(), u.PassHash(), u.CreatedAt(), u.UpdatedAt())

	require.NoError(t, err)
}
