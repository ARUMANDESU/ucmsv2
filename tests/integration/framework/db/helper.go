package db

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/ucmsv2/ucms-backend/internal/adapters/repos/postgres"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/group"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/staffinvitation"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/majors"
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
		"staff_invitations",
		"registrations",
		"staffs",
		"students",
		"groups",
		"users",
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

func (h *Helper) RequireStaffExists(t *testing.T, id user.ID) *user.StaffAssertions {
	t.Helper()

	staff, err := h.staff.GetStaffByID(t.Context(), id)
	require.NoError(t, err, "staff not found for user_id: %s", id)

	return user.NewStaffAssertions(staff)
}

func (h *Helper) RequireStaffExistsByEmail(t *testing.T, email string) *user.StaffAssertions {
	t.Helper()

	staff, err := h.staff.GetStaffByEmail(t.Context(), email)
	require.NoError(t, err, "staff not found for email: %s", email)

	return user.NewStaffAssertions(staff)
}

func (h *Helper) RequireStaffNotExistsByEmail(t *testing.T, email string) {
	t.Helper()

	var count int
	err := h.pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM staffs s JOIN users u ON s.user_id = u.id WHERE u.email = $1", email).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "expected no staff for email %s", email)
}

func (h *Helper) RequireStaffInvitationExists(t *testing.T, id staffinvitation.ID) *staffinvitation.Assertion {
	t.Helper()

	invitation, err := h.staffInvitation.GetStaffInvitationByID(t.Context(), id)
	require.NoError(t, err, "staff invitation not found for id: %s", id)

	return staffinvitation.NewAssertion(t, invitation)
}

func (h *Helper) RequireStaffInvitationExistsByCode(t *testing.T, code string) *staffinvitation.Assertion {
	t.Helper()

	invitation, err := h.staffInvitation.GetStaffInvitationByCode(t.Context(), code)
	require.NoError(t, err, "staff invitation not found for code: %s", code)

	return staffinvitation.NewAssertion(t, invitation)
}

func (h *Helper) RequireLatestStaffInvitationByCreatorID(t *testing.T, creatorID user.ID) *staffinvitation.Assertion {
	t.Helper()

	invitation, err := h.staffInvitation.GetLatestStaffInvitationByCreatorID(t.Context(), creatorID)
	require.NoError(t, err, "no staff invitation found for creator_id: %s", creatorID)

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

	err := h.user.SaveUser(t.Context(), u)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if err := h.user.UpdateUser(t.Context(), u.ID(), func(ctx context.Context, dbU *user.User) error {
				*dbU = *u
				return nil
			}); err == nil {
				return
			}
		}
		t.Fatalf("failed to upsert user: %v", err)
	}
}

func (h *Helper) SeedStudent(t *testing.T, student *user.Student) {
	t.Helper()
	require.NoError(t, h.student.SaveStudent(t.Context(), student))
}

func (h *Helper) SeedGroup(t *testing.T, groupID group.ID, name string, year string, major majors.Major) {
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

func (h *Helper) SeedStaffInvitation(t *testing.T, invitation *staffinvitation.StaffInvitation) {
	t.Helper()
	require.NoError(t, h.staffInvitation.SaveStaffInvitation(t.Context(), invitation))
}
