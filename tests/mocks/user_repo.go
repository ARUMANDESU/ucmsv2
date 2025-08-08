package mocks

import (
	"context"
	"sync"
	"testing"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
)

type UserRepo struct {
	dbbyID    map[user.ID]*user.User
	dbbyEmail map[string]*user.User
	events    []event.Event
	mu        sync.Mutex
}

func NewUserRepo() *UserRepo {
	return &UserRepo{
		dbbyID:    make(map[user.ID]*user.User),
		dbbyEmail: make(map[string]*user.User),
	}
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.dbbyEmail[email]; ok {
		return u, nil
	}
	return nil, repos.ErrNotFound
}

func (r *UserRepo) GetUserByID(ctx context.Context, id user.ID) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.dbbyID[id]; ok {
		return u, nil
	}
	return nil, repos.ErrNotFound
}

func (r *UserRepo) SeedUser(t *testing.T, u *user.User) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbbyID[u.ID()]; exists {
		t.Fatalf("user with ID %s already exists", u.ID())
	}

	if _, exists := r.dbbyEmail[u.Email()]; exists {
		t.Fatalf("user with email %s already exists", u.Email())
	}

	r.dbbyID[u.ID()] = u
	r.dbbyEmail[u.Email()] = u
}
