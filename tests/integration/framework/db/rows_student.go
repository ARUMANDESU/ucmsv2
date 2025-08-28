package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type StudentRow struct {
	UserID    uuid.UUID
	GroupID   uuid.UUID
	GroupName string
	Year      string
	Major     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StudentAssertion struct {
	t   *testing.T
	row StudentRow
}

func (a *StudentAssertion) AssertInGroupID(groupID uuid.UUID) *StudentAssertion {
	a.t.Helper()
	assert.Equal(a.t, groupID, a.row.GroupID, "unexpected group ID")
	return a
}

func (a *StudentAssertion) AssertInGroup(groupName string) *StudentAssertion {
	a.t.Helper()
	assert.Equal(a.t, groupName, a.row.GroupName, "unexpected group name")
	return a
}

func (a *StudentAssertion) AssertMajor(major string) *StudentAssertion {
	a.t.Helper()
	assert.Equal(a.t, major, a.row.Major, "unexpected major")
	return a
}
