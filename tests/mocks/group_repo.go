package mocks

import (
	"context"
	"sync"
	"testing"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type GroupRepo struct {
	*EventRepo
	dbByID   map[group.ID]*group.Group
	dbByName map[string]*group.Group
	mu       sync.Mutex
}

func NewGroupRepo() *GroupRepo {
	return &GroupRepo{
		EventRepo: NewEventRepo(),
		dbByID:    make(map[group.ID]*group.Group),
		dbByName:  make(map[string]*group.Group),
		mu:        sync.Mutex{},
	}
}

func (r *GroupRepo) GetGroupByID(_ context.Context, id group.ID) (*group.Group, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group, exists := r.dbByID[id]; exists {
		return group, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *GroupRepo) GetGroupByName(_ context.Context, name string) (*group.Group, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group, exists := r.dbByName[name]; exists {
		return group, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *GroupRepo) SeedGroup(ctx context.Context, group *group.Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group == nil {
		return errorx.NewValidationFieldFailed("group").WithArgs(map[string]any{"Field": "group"})
	}

	if _, exists := r.dbByID[group.ID()]; exists {
		return errorx.NewDuplicateEntryWithField("group", "id")
	}

	if _, exists := r.dbByName[group.Name()]; exists {
		return errorx.NewDuplicateEntryWithField("group", "name")
	}

	r.dbByID[group.ID()] = group
	r.dbByName[group.Name()] = group

	return nil
}

func (r *GroupRepo) AssertGroupNotExists(t *testing.T, id group.ID) *GroupRepo {
	t.Helper()
	_, err := r.GetGroupByID(context.Background(), id)
	if err == nil {
		t.Errorf("Expected group with ID %s to not exist, but it does", id)
		return r
	}
	if !errorx.IsNotFound(err) {
		t.Errorf("Expected group with ID %s to not exist, but got unexpected error: %v", id, err)
		return r
	}
	return r
}

func (r *GroupRepo) AssertGroupNotExistsByName(t *testing.T, name string) *GroupRepo {
	t.Helper()
	_, err := r.GetGroupByName(context.Background(), name)
	if err == nil {
		t.Errorf("Expected group with name %s to not exist, but it does", name)
		return r
	}
	if !errorx.IsNotFound(err) {
		t.Errorf("Expected group with name %s to not exist, but got unexpected error: %v", name, err)
		return r
	}
	return r
}

func (r *GroupRepo) RequireGroupByID(t *testing.T, id group.ID) *group.GroupAssertion {
	t.Helper()
	g, err := r.GetGroupByID(context.Background(), id)
	if err != nil {
		t.Fatalf("Failed to get group by ID %s: %v", id, err)
	}
	return group.NewGroupAssertion(g)
}

func (r *GroupRepo) RequireGroupByName(t *testing.T, name string) *group.GroupAssertion {
	t.Helper()
	g, err := r.GetGroupByName(context.Background(), name)
	if err != nil {
		t.Fatalf("Failed to get group by name %s: %v", name, err)
	}
	return group.NewGroupAssertion(g)
}
