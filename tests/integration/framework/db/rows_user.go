package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type UserRow struct {
	ID        string
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

func (a *UserAssertion) HasRole(expected string) *UserAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.row.RoleName, "unexpected user role")
	return a
}

func (a *UserAssertion) HasFullName(firstName, lastName string) *UserAssertion {
	a.t.Helper()
	assert.Equal(a.t, firstName, a.row.FirstName, "unexpected first name")
	assert.Equal(a.t, lastName, a.row.LastName, "unexpected last name")
	return a
}

func (a *UserAssertion) IsStudent() *StudentAssertion {
	a.t.Helper()
	return a.db.AssertStudentExists(a.t, a.row.ID)
}
