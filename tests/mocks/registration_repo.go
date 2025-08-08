package mocks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
)

type RegistrationRepo struct {
	dbbyEmail map[string]*registration.Registration
	dbbyID    map[registration.ID]*registration.Registration
	dbbyCode  map[string]*registration.Registration
	events    []event.Event
	eventsMu  sync.Mutex
	eventCh   chan event.Event
	mu        sync.Mutex
}

func NewRegistrationRepo() *RegistrationRepo {
	return &RegistrationRepo{
		dbbyEmail: make(map[string]*registration.Registration),
		dbbyID:    make(map[registration.ID]*registration.Registration),
		dbbyCode:  make(map[string]*registration.Registration),
		events:    []event.Event{},
		eventCh:   make(chan event.Event, 100),
		mu:        sync.Mutex{},
	}
}

func (r *RegistrationRepo) GetRegistrationByEmail(ctx context.Context, email string) (*registration.Registration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if reg, exists := r.dbbyEmail[email]; exists {
		return reg, nil
	}
	return nil, repos.ErrNotFound
}

func (r *RegistrationRepo) SaveRegistration(ctx context.Context, reg *registration.Registration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if reg == nil {
		return fmt.Errorf("%w: %w", repos.ErrInvalidInput, errors.New("registration cannot be nil"))
	}

	if _, exists := r.dbbyEmail[reg.Email()]; exists {
		return repos.ErrAlreadyExists
	}

	if _, exists := r.dbbyID[reg.ID()]; exists {
		return repos.ErrAlreadyExists
	}

	r.dbbyEmail[reg.Email()] = reg
	r.dbbyID[reg.ID()] = reg
	r.dbbyCode[reg.VerificationCode()] = reg

	r.eventsMu.Lock()
	r.events = append(r.events, reg.GetUncommittedEvents()...)
	r.eventsMu.Unlock()
	for _, event := range reg.GetUncommittedEvents() {
		select {
		case r.eventCh <- event:
		default:
			// If the channel is full, we skip sending the event to avoid blocking.
		}
	}

	return nil
}

func (r *RegistrationRepo) UpdateRegistration(
	ctx context.Context,
	id registration.ID,
	fn func(context.Context, *registration.Registration) error,
) error {
	if fn == nil {
		return fmt.Errorf("%w: %w", repos.ErrInvalidInput, errors.New("update function cannot be nil"))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.dbbyID[id]
	if !exists {
		return repos.ErrNotFound
	}

	if err := fn(ctx, reg); err != nil {
		return fmt.Errorf("failed to apply update function: %w", err)
	}

	r.dbbyID[id] = reg
	r.dbbyEmail[reg.Email()] = reg
	r.dbbyCode[reg.VerificationCode()] = reg

	r.eventsMu.Lock()
	r.events = append(r.events, reg.GetUncommittedEvents()...)
	r.eventsMu.Unlock()
	for _, event := range reg.GetUncommittedEvents() {
		select {
		case r.eventCh <- event:
		default:
			// If the channel is full, we skip sending the event to avoid blocking.
		}
	}

	return nil
}

func (r *RegistrationRepo) UpdateRegistrationByEmail(
	ctx context.Context,
	email string,
	fn func(context.Context, *registration.Registration) error,
) error {
	if fn == nil {
		return fmt.Errorf("%w: %w", repos.ErrInvalidInput, errors.New("update function cannot be nil"))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.dbbyEmail[email]
	if !exists {
		return repos.ErrNotFound
	}

	if err := fn(ctx, reg); err != nil {
		return fmt.Errorf("failed to apply update function: %w", err)
	}

	r.dbbyEmail[email] = reg
	r.dbbyID[reg.ID()] = reg
	r.dbbyCode[reg.VerificationCode()] = reg

	r.eventsMu.Lock()
	r.events = append(r.events, reg.GetUncommittedEvents()...)
	r.eventsMu.Unlock()
	for _, event := range reg.GetUncommittedEvents() {
		select {
		case r.eventCh <- event:
		default:
			// If the channel is full, we skip sending the event to avoid blocking.
		}
	}

	return nil
}

func (r *RegistrationRepo) EventChannel() <-chan event.Event {
	return r.eventCh
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
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	for _, ev := range r.events {
		if fmt.Sprintf("%T", ev) == fmt.Sprintf("%T", e) {
			t.Errorf("expected event %T to not exist, but it does", e)
			return r
		}
	}

	return r
}

func (r *RegistrationRepo) AssertEventCount(t *testing.T, count int) *RegistrationRepo {
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	if len(r.events) != count {
		t.Errorf("expected %d events, but got %d", count, len(r.events))
	}

	return r
}

func RequireEventExists[T event.Event](t *testing.T, r *RegistrationRepo, e T) T {
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	tnil := *new(T)

	for _, ev := range r.events {
		if fmt.Sprintf("%T", ev) == fmt.Sprintf("%T", e) {
			if ev == nil {
				t.Errorf("expected event %T to not be nil, but it is", e)
				return tnil
			}
			header := ev.GetEventHeader()
			assert.NotEmpty(t, header, "event header should not be empty")
			return ev.(T)
		}
	}

	t.Fatalf("event %T not found in repository", e)

	return tnil
}
