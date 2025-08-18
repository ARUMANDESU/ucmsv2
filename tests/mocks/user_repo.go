package mocks

import (
	"context"
	"sync"
	"testing"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type UserRepo struct {
	dbbyID    map[user.Barcode]*user.User
	dbbyEmail map[string]*user.User
	events    []event.Event
	mu        sync.Mutex
}

func NewUserRepo() *UserRepo {
	return &UserRepo{
		dbbyID:    make(map[user.Barcode]*user.User),
		dbbyEmail: make(map[string]*user.User),
	}
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.dbbyEmail[email]; ok {
		return u, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *UserRepo) GetUserByBarcode(ctx context.Context, barcode user.Barcode) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.dbbyID[barcode]; ok {
		return u, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *UserRepo) SeedUser(t *testing.T, u *user.User) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbbyID[u.Barcode()]; exists {
		t.Fatalf("user with barcode %s already exists", u.Barcode())
	}

	if _, exists := r.dbbyEmail[u.Email()]; exists {
		t.Fatalf("user with email %s already exists", u.Email())
	}

	r.dbbyID[u.Barcode()] = u
	r.dbbyEmail[u.Email()] = u
}
