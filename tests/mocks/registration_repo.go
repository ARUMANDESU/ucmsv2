package mocks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
)

type RegistrationRepo struct {
	*EventRepo
	dbbyEmail map[string]*registration.Registration
	dbbyID    map[registration.ID]*registration.Registration
	dbbyCode  map[string]*registration.Registration
	mu        sync.Mutex
}

func NewRegistrationRepo() *RegistrationRepo {
	return &RegistrationRepo{
		EventRepo: NewEventRepo(),
		dbbyEmail: make(map[string]*registration.Registration),
		dbbyID:    make(map[registration.ID]*registration.Registration),
		dbbyCode:  make(map[string]*registration.Registration),
		mu:        sync.Mutex{},
	}
}

func (r *RegistrationRepo) GetRegistrationByEmail(ctx context.Context, email string) (*registration.Registration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if reg, exists := r.dbbyEmail[email]; exists {
		return reg, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *RegistrationRepo) SaveRegistration(ctx context.Context, reg *registration.Registration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if reg == nil {
		return errors.New("registration cannot be nil")
	}

	if _, exists := r.dbbyEmail[reg.Email()]; exists {
		return errorx.NewDuplicateEntry()
	}

	if _, exists := r.dbbyID[reg.ID()]; exists {
		return errorx.NewDuplicateEntry()
	}

	r.dbbyEmail[reg.Email()] = reg
	r.dbbyID[reg.ID()] = reg
	r.dbbyCode[reg.VerificationCode()] = reg

	r.appendEvents(reg.GetUncommittedEvents()...)

	return nil
}

func (r *RegistrationRepo) UpdateRegistration(
	ctx context.Context,
	id registration.ID,
	fn func(context.Context, *registration.Registration) error,
) error {
	if fn == nil {
		return errors.New("update function cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.dbbyID[id]
	if !exists {
		return errorx.NewNotFound()
	}

	fnerr := fn(ctx, reg)
	if fnerr != nil && !errorx.IsPersistable(fnerr) {
		return fmt.Errorf("failed to apply update function: %w", fnerr)
	}

	r.dbbyID[id] = reg
	r.dbbyEmail[reg.Email()] = reg
	r.dbbyCode[reg.VerificationCode()] = reg

	r.appendEvents(reg.GetUncommittedEvents()...)

	if fnerr != nil && errorx.IsPersistable(fnerr) {
		return fmt.Errorf("failed to apply update function: %w", fnerr)
	}
	return nil
}

func (r *RegistrationRepo) UpdateRegistrationByEmail(
	ctx context.Context,
	email string,
	fn func(context.Context, *registration.Registration) error,
) error {
	if fn == nil {
		return errors.New("update function cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.dbbyEmail[email]
	if !exists {
		return errorx.NewNotFound()
	}

	fnerr := fn(ctx, reg)
	if fnerr != nil && !errorx.IsPersistable(fnerr) {
		return fmt.Errorf("failed to apply update function: %w", fnerr)
	}

	r.dbbyEmail[email] = reg
	r.dbbyID[reg.ID()] = reg
	r.dbbyCode[reg.VerificationCode()] = reg

	r.appendEvents(reg.GetUncommittedEvents()...)

	if fnerr != nil && errorx.IsPersistable(fnerr) {
		return fmt.Errorf("failed to apply update function: %w", fnerr)
	}
	return nil
}

func (r *RegistrationRepo) SeedRegistration(t *testing.T, reg *registration.Registration) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbbyEmail[reg.Email()]; exists {
		t.Fatalf("registration with email %s already exists", reg.Email())
	}

	if _, exists := r.dbbyID[reg.ID()]; exists {
		t.Fatalf("registration with ID %s already exists", reg.ID())
	}

	r.dbbyEmail[reg.Email()] = reg
	r.dbbyID[reg.ID()] = reg
	r.dbbyCode[reg.VerificationCode()] = reg

	r.appendEvents(reg.GetUncommittedEvents()...)
}

func (r *RegistrationRepo) AssertRegistrationExistsByEmail(t *testing.T, email string) *registration.RegistrationAssertion {
	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.dbbyEmail[email]
	if !exists {
		t.Errorf("expected registration with email %s to exist, but it does not", email)
		return nil
	}

	return registration.NewRegistrationAssertion(reg)
}

func (r *RegistrationRepo) AssertRegistrationNotExistsByEmail(t *testing.T, email string) *RegistrationRepo {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbbyEmail[email]; exists {
		t.Errorf("expected registration with email %s to not exist, but it does", email)
		return r
	}

	return r
}

func (r *RegistrationRepo) AssertRegistrationExistsByID(t *testing.T, id registration.ID) *registration.RegistrationAssertion {
	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.dbbyID[id]
	if !exists {
		t.Errorf("expected registration with ID %s to exist, but it does not", id)
		return nil
	}
	return registration.NewRegistrationAssertion(reg)
}

func (r *RegistrationRepo) AssertRegistrationNotExistsByID(t *testing.T, id registration.ID) *RegistrationRepo {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbbyID[id]; exists {
		t.Errorf("expected registration with ID %s to not exist, but it does", id)
		return r
	}
	return r
}

func (r *RegistrationRepo) AssertEventNotExists(t *testing.T, e event.Event) *RegistrationRepo {
	r.EventRepo.AssertEventNotExists(t, e)
	return r
}

func (r *RegistrationRepo) AssertEventCount(t *testing.T, count int) *RegistrationRepo {
	r.EventRepo.AssertEventCount(t, count)
	return r
}
