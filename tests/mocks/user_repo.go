package mocks

import (
	"context"
	"sync"
	"testing"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type UserRepo struct {
	dbbyID      map[user.ID]*user.User
	dbbyEmail   map[string]*user.User
	dbbyBarcode map[user.Barcode]*user.User
	// events      []event.Event
	mu sync.Mutex
}

func NewUserRepo() *UserRepo {
	return &UserRepo{
		dbbyID:      make(map[user.ID]*user.User),
		dbbyEmail:   make(map[string]*user.User),
		dbbyBarcode: make(map[user.Barcode]*user.User),
	}
}

func (r *UserRepo) GetUserByID(ctx context.Context, id user.ID) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.dbbyID[id]; ok {
		return u, nil
	}
	return nil, errorx.NewNotFound()
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

	if u, ok := r.dbbyBarcode[barcode]; ok {
		return u, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *UserRepo) IsUserExists(
	ctx context.Context,
	email, username string,
	barcode user.Barcode,
) (emailExists, usernameExists, barcodeExists bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, emailExists = r.dbbyEmail[email]
	_, barcodeExists = r.dbbyBarcode[barcode]
	for _, u := range r.dbbyID {
		if u.Username() == username {
			usernameExists = true
			break
		}
	}
	return emailExists, usernameExists, barcodeExists, nil
}

func (r *UserRepo) SeedUser(t *testing.T, u *user.User) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if u == nil {
		t.Fatal("cannot seed nil user")
	}

	if _, exists := r.dbbyID[u.ID()]; exists {
		t.Fatalf("user with ID %s already exists", u.ID().String())
	}

	if _, exists := r.dbbyBarcode[u.Barcode()]; exists {
		t.Fatalf("user with barcode %s already exists", u.Barcode())
	}

	if _, exists := r.dbbyEmail[u.Email()]; exists {
		t.Fatalf("user with email %s already exists", u.Email())
	}

	r.dbbyID[u.ID()] = u
	r.dbbyBarcode[u.Barcode()] = u
	r.dbbyEmail[u.Email()] = u
}
