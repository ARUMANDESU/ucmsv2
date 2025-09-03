package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos/postgres"
	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
)

type Helper struct {
	pool            *pgxpool.Pool
	group           *postgres.GroupRepo
	user            *postgres.UserRepo
	student         *postgres.StudentRepo
	staff           *postgres.StaffRepo
	staffInvitation *postgres.StaffInvitationRepo
	registration    *postgres.RegistrationRepo
}

type Args struct {
	Pool            *pgxpool.Pool
	Group           *postgres.GroupRepo
	User            *postgres.UserRepo
	Student         *postgres.StudentRepo
	Staff           *postgres.StaffRepo
	StaffInvitation *postgres.StaffInvitationRepo
	Registration    *postgres.RegistrationRepo
}

func NewHelper(args Args) *Helper {
	if args.Pool == nil {
		panic("pgxpool.Pool is required")
	}
	if args.User == nil {
		args.User = postgres.NewUserRepo(args.Pool, nil, nil)
	}
	if args.Student == nil {
		args.Student = postgres.NewStudentRepo(args.Pool, nil, nil)
	}
	if args.Group == nil {
		args.Group = postgres.NewGroupRepo(args.Pool, nil, nil)
	}
	if args.Staff == nil {
		args.Staff = postgres.NewStaffRepo(args.Pool, nil, nil)
	}
	if args.StaffInvitation == nil {
		args.StaffInvitation = postgres.NewStaffInvitationRepo(args.Pool, nil, nil)
	}
	if args.Registration == nil {
		args.Registration = postgres.NewRegistrationRepo(args.Pool, nil, nil)
	}

	return &Helper{
		pool:            args.Pool,
		user:            args.User,
		student:         args.Student,
		group:           args.Group,
		staff:           args.Staff,
		staffInvitation: args.StaffInvitation,
		registration:    args.Registration,
	}
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
		"staff_invitations",
	}

	ctx := context.Background()
	for _, table := range tables {
		_, err := h.pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "failed to truncate table %s", table)
	}
}

func (h *Helper) RequireRegistrationExists(t *testing.T, email string) *registration.RegistrationAssertion {
	t.Helper()

	reg, err := h.registration.GetRegistrationByEmail(t.Context(), email)
	require.NoError(t, err, "registration not found for email: %s", email)

	return registration.NewRegistrationAssertion(reg)
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

	return exists
}

func (h *Helper) RequireUserExists(t *testing.T, email string) *user.UserAssertions {
	t.Helper()

	u, err := h.user.GetUserByEmail(t.Context(), email)
	require.NoError(t, err, "user not found for email: %s", email)

	return user.NewUserAssertions(t, u)
}

func (h *Helper) RequireUserNotExists(t *testing.T, email string) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "expected no user for email %s", email)
}

func (h *Helper) RequireStudentExists(t *testing.T, id user.ID) *user.StudentAssertions {
	t.Helper()

	student, err := h.student.GetStudentByID(t.Context(), id)
	require.NoError(t, err, "student not found for user_id: %s", id)

	return user.NewStudentAssertions(student)
}

func (h *Helper) RequireStudentExistsByEmail(t *testing.T, email string) *user.StudentAssertions {
	t.Helper()

	student, err := h.student.GetStudentByEmail(t.Context(), email)
	require.NoError(t, err, "student not found for email: %s", email)

	return user.NewStudentAssertions(student)
}

func (h *Helper) RequireStaffInvitationExists(t *testing.T, code string) *staffinvitation.Assertion {
	t.Helper()

	invitation, err := h.staffInvitation.GetStaffInvitationByCode(t.Context(), code)
	require.NoError(t, err, "staff invitation not found for code: %s", code)

	return staffinvitation.NewAssertion(t, invitation)
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
	require.NoError(t, h.registration.SaveRegistration(t.Context(), r))
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

func (h *Helper) SeedStudent(t *testing.T, student *user.Student) {
	t.Helper()
	require.NoError(t, h.student.SaveStudent(t.Context(), student))
}

func (h *Helper) SeedGroup(t *testing.T, groupID group.ID, name string, year string, major major.Major) {
	t.Helper()
	g := group.Rehydrate(group.RehydrateArgs{
		ID:        groupID,
		Name:      name,
		Major:     major,
		Year:      year,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	require.NoError(t, h.group.SaveGroup(t.Context(), g))
}

func (h *Helper) SeedStaff(t *testing.T, staff *user.Staff) {
	t.Helper()
	require.NoError(t, h.staff.SaveStaff(t.Context(), staff))
}
