package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type UserRow struct {
	ID        uuid.UUID
	Barcode   string
	Email     string
	FirstName string
	LastName  string
	RoleID    int16
	RoleName  string
	AvatarURL string
	PassHash  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserAssertion struct {
	row UserRow
	t   *testing.T
	db  *Helper
}

func (a *UserAssertion) AssertRole(expected role.Global) *UserAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected.String(), a.row.RoleName, "unexpected user role")
	return a
}

func (a *UserAssertion) AssertFullName(firstName, lastName string) *UserAssertion {
	a.t.Helper()
	assert.Equal(a.t, firstName, a.row.FirstName, "unexpected first name")
	assert.Equal(a.t, lastName, a.row.LastName, "unexpected last name")
	return a
}

func (a *UserAssertion) AssertFirstName(expected string) *UserAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.row.FirstName, "unexpected first name")
	return a
}

func (a *UserAssertion) AssertIsStudent() *StudentAssertion {
	a.t.Helper()
	return a.db.RequireStudentExists(a.t, user.ID(a.row.ID))
}
