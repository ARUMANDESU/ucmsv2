package db

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
)

type Helper struct {
	pool *pgxpool.Pool
}

func NewHelper(pool *pgxpool.Pool) *Helper {
	return &Helper{pool: pool}
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

func (h *Helper) RequireRegistrationExists(t *testing.T, email string) *RegistrationAssertion {
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

func (h *Helper) RequireRegistrationNotExists(t *testing.T, email string) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM registrations WHERE email = $1", email).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "expected no registration for email %s, but found %d", email, count)
}

func (h *Helper) RequireRegistrationCount(t *testing.T, expected int) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM registrations").Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, expected, count, "unexpected registration count")
}

func (h *Helper) CheckUserExists(t *testing.T, email string) bool {
	t.Helper()

	var exists bool
	err := h.pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	require.NoError(t, err)

	rows, err := h.pool.Query(t.Context(), "SELECT id, barcode, email, first_name, last_name, role_id FROM users")
	assert.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var barcode, email, firstName, lastName string
		var roleID int16
		err := rows.Scan(&id, &barcode, &email, &firstName, &lastName, &roleID)
		require.NoError(t, err)
		slog.Info("user:", slog.String("id", id.String()), slog.String("barcode", barcode), slog.String("email", email),
			slog.String("first_name", firstName), slog.String("last_name", lastName),
			slog.Int("role_id", int(roleID)),
		)
	}

	require.NoError(t, err)
	return exists
}

func (h *Helper) RequireUserExists(t *testing.T, email string) *UserAssertion {
	t.Helper()

	var u UserRow
	err := h.pool.QueryRow(context.Background(), `
        SELECT u.id, u.barcode, u.username, u.email, u.first_name, u.last_name, u.role_id, 
               u.avatar_url, u.pass_hash, u.created_at, u.updated_at,
               gr.name as role_name
        FROM users u
        JOIN global_roles gr ON u.role_id = gr.id
        WHERE u.email = $1
    `, email).Scan(
		&u.ID, &u.Barcode, &u.Username, &u.Email, &u.FirstName, &u.LastName, &u.RoleID,
		&u.AvatarURL, &u.PassHash, &u.CreatedAt, &u.UpdatedAt, &u.RoleName,
	)

	require.NoError(t, err, "user not found for email: %s", email)

	return &UserAssertion{t: t, row: u, db: h}
}

func (h *Helper) RequireUserNotExists(t *testing.T, email string) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "expected no user for email %s", email)
}

func (h *Helper) RequireStudentExists(t *testing.T, userID user.ID) *StudentAssertion {
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

func (h *Helper) CheckGroupExists(t *testing.T, groupID group.ID) bool {
	t.Helper()

	var exists bool
	err := h.pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)", groupID).Scan(&exists)

	require.NoError(t, err)
	return exists
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
        INSERT INTO users (id, barcode, username, email, role_id, first_name, last_name, 
                          avatar_url, pass_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (id) DO UPDATE SET
            barcode = EXCLUDED.barcode,
            username = EXCLUDED.username,
            email = EXCLUDED.email,
            role_id = EXCLUDED.role_id,
            first_name = EXCLUDED.first_name,
            last_name = EXCLUDED.last_name,
            avatar_url = EXCLUDED.avatar_url,
            pass_hash = EXCLUDED.pass_hash,
            updated_at = EXCLUDED.updated_at
    `, u.ID(), u.Barcode().String(), u.Username(), u.Email(), roleID, u.FirstName(), u.LastName(),
		u.AvatarUrl(), u.PassHash(), u.CreatedAt(), u.UpdatedAt())

	require.NoError(t, err)
}

func (h *Helper) SeedStudent(t *testing.T, userID user.ID, groupID group.ID) {
	t.Helper()

	_, err := h.pool.Exec(context.Background(), `
        INSERT INTO students (user_id, group_id, created_at, updated_at)
        VALUES ($1, $2, NOW(), NOW())
    `, userID, groupID)

	require.NoError(t, err)
}

func (h *Helper) SeedGroup(t *testing.T, groupID group.ID, name string, year string, major major.Major) {
	t.Helper()

	_, err := h.pool.Exec(context.Background(), `
        INSERT INTO groups (id, name, year, major, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `, groupID, name, year, major)

	require.NoError(t, err)
}
