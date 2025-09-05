package group

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/majors"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
)

const (
	MinNameLength = 2
	MaxNameLength = 100
	MinYearLength = 1
	MaxYearLength = 3
)

var YearPattern = regexp.MustCompile(`^\d{1,3}$`)

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

type Group struct {
	id        ID
	name      string
	major     majors.Major
	year      string
	createdAt time.Time
	updatedAt time.Time
}

func NewGroup(name, year string, m majors.Major) (*Group, error) {
	const op = "group.NewGroup"
	err := validation.Validate(name, validation.Required, validation.Length(MinNameLength, MaxNameLength))
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}
	err = validation.Validate(
		year,
		validation.Required,
		validation.Length(MinYearLength, MaxYearLength),
		validation.Match(YearPattern).Error("validation_"),
	)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}
	if !majors.IsValid(m) {
		return nil, errorx.Wrap(majors.ErrInvalidMajor, op)
	}

	now := time.Now().UTC()

	return &Group{
		id:        NewID(),
		name:      name,
		major:     m,
		year:      year,
		createdAt: now,
		updatedAt: now,
	}, nil
}

type RehydrateArgs struct {
	ID        ID
	Name      string
	Major     majors.Major
	Year      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func Rehydrate(args RehydrateArgs) *Group {
	return &Group{
		id:        args.ID,
		name:      args.Name,
		major:     args.Major,
		year:      args.Year,
		createdAt: args.CreatedAt,
		updatedAt: args.UpdatedAt,
	}
}

func (g *Group) ID() ID {
	return g.id
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) Major() majors.Major {
	return g.major
}

func (g *Group) Year() string {
	return g.year
}

func (g *Group) CreatedAt() time.Time {
	return g.createdAt
}

func (g *Group) UpdatedAt() time.Time {
	return g.updatedAt
}

type GroupAssertion struct {
	group *Group
}

func NewGroupAssertion(g *Group) *GroupAssertion {
	if g == nil {
		return nil
	}
	return &GroupAssertion{group: g}
}

func (a *GroupAssertion) AssertID(t *testing.T, expected ID) *GroupAssertion {
	t.Helper()
	assert.Equal(t, expected, a.group.ID(), "Expected group ID to be %s, got %s", expected, a.group.ID())
	return a
}

func (a *GroupAssertion) AssertName(t *testing.T, expected string) *GroupAssertion {
	t.Helper()
	assert.Equal(t, expected, a.group.Name(), "Expected group name to be %s, got %s", expected, a.group.Name())
	return a
}

func (a *GroupAssertion) AssertMajor(t *testing.T, expected majors.Major) *GroupAssertion {
	t.Helper()
	assert.Equal(t, expected, a.group.Major(), "Expected group major to be %s, got %s", expected, a.group.Major())
	return a
}

func (a *GroupAssertion) AssertYear(t *testing.T, expected string) *GroupAssertion {
	t.Helper()
	assert.Equal(t, expected, a.group.Year(), "Expected group year to be %s, got %s", expected, a.group.Year())
	return a
}

func (a *GroupAssertion) AssertCreatedAt(t *testing.T, expected time.Time) *GroupAssertion {
	t.Helper()
	assert.WithinDuration(
		t,
		expected,
		a.group.CreatedAt(),
		time.Second,
		"Expected group created at to be within 1 second of %s, got %s",
		expected,
		a.group.CreatedAt(),
	)
	return a
}

func (a *GroupAssertion) AssertUpdatedAt(t *testing.T, expected time.Time) *GroupAssertion {
	t.Helper()
	assert.WithinDuration(
		t,
		expected,
		a.group.UpdatedAt(),
		time.Second,
		"Expected group updated at to be within 1 second of %s, got %s",
		expected,
		a.group.UpdatedAt(),
	)
	return a
}
