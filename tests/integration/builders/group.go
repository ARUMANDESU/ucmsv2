package builders

import (
	"time"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
)

type GroupFactory struct{}

func (f *GroupFactory) DefaultSEGroup() *group.Group {
	return NewGroupBuilder().Build()
}

func (f *GroupFactory) ITGroup() *group.Group {
	return NewGroupBuilder().
		WithID(group.ID(fixtures.ITGroup.ID)).
		WithName(fixtures.ITGroup.Name).
		WithMajor(fixtures.ITGroup.Major).
		WithYear(fixtures.ITGroup.Year).
		Build()
}

func (f *GroupFactory) CSGroup() *group.Group {
	return NewGroupBuilder().
		WithID(group.ID(fixtures.CSGroup.ID)).
		WithName(fixtures.CSGroup.Name).
		WithMajor(fixtures.CSGroup.Major).
		WithYear(fixtures.CSGroup.Year).
		Build()
}

type GroupBuilder struct {
	id        group.ID
	name      string
	major     major.Major
	year      string
	createdAt time.Time
	updatedAt time.Time
}

func NewGroupBuilder() *GroupBuilder {
	return &GroupBuilder{
		id:        group.ID(fixtures.SEGroup.ID),
		name:      fixtures.SEGroup.Name,
		major:     fixtures.SEGroup.Major,
		year:      fixtures.SEGroup.Year,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

func (b *GroupBuilder) WithID(id group.ID) *GroupBuilder {
	b.id = id
	return b
}

func (b *GroupBuilder) WithName(name string) *GroupBuilder {
	b.name = name
	return b
}

func (b *GroupBuilder) WithMajor(major major.Major) *GroupBuilder {
	b.major = major
	return b
}

func (b *GroupBuilder) WithYear(year string) *GroupBuilder {
	b.year = year
	return b
}

func (b *GroupBuilder) WithCreatedAt(createdAt time.Time) *GroupBuilder {
	b.createdAt = createdAt
	return b
}

func (b *GroupBuilder) WithUpdatedAt(updatedAt time.Time) *GroupBuilder {
	b.updatedAt = updatedAt
	return b
}

func (b *GroupBuilder) Build() *group.Group {
	return group.Rehydrate(group.RehydrateArgs{
		ID:        b.id,
		Name:      b.name,
		Major:     b.major,
		Year:      b.year,
		CreatedAt: b.createdAt,
		UpdatedAt: b.updatedAt,
	})
}
