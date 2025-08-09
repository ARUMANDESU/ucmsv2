package mocks

import (
	"context"
	"sync"
	"testing"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos"
	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
)

type StudentRepo struct {
	*EventRepo
	dbByEmail map[string]*user.Student
	dbByID    map[user.ID]*user.Student
	mu        sync.Mutex
}

func NewStudentRepo() *StudentRepo {
	return &StudentRepo{
		EventRepo: NewEventRepo(),
		dbByEmail: make(map[string]*user.Student),
		dbByID:    make(map[user.ID]*user.Student),
		mu:        sync.Mutex{},
	}
}

func (r *StudentRepo) GetStudentByEmail(ctx context.Context, email string) (*user.Student, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if student, exists := r.dbByEmail[email]; exists {
		return student, nil
	}
	return nil, repos.ErrNotFound
}

func (r *StudentRepo) GetStudentByID(ctx context.Context, id user.ID) (*user.Student, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if student, exists := r.dbByID[id]; exists {
		return student, nil
	}
	return nil, repos.ErrNotFound
}

func (r *StudentRepo) SaveStudent(ctx context.Context, student *user.Student) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if student == nil {
		return repos.ErrInvalidInput
	}

	if _, exists := r.dbByEmail[student.User().Email()]; exists {
		return repos.ErrAlreadyExists
	}

	if _, exists := r.dbByID[student.User().ID()]; exists {
		return repos.ErrAlreadyExists
	}

	r.dbByEmail[student.User().Email()] = student
	r.dbByID[student.User().ID()] = student

	r.EventRepo.appendEvents(student.GetUncommittedEvents()...)

	return nil
}

func (r *StudentRepo) SeedStudent(t *testing.T, student *user.Student) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbByID[student.User().ID()]; exists {
		t.Fatalf("student with ID %s already exists", student.User().ID())
	}

	if _, exists := r.dbByEmail[student.User().Email()]; exists {
		t.Fatalf("student with email %s already exists", student.User().Email())
	}

	r.dbByID[student.User().ID()] = student
	r.dbByEmail[student.User().Email()] = student
	r.EventRepo.appendEvents(student.GetUncommittedEvents()...)
}

func (r *StudentRepo) EventChannel() <-chan event.Event {
	return r.eventCh
}

func (r *StudentRepo) Events() []event.Event {
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	eventsCopy := make([]event.Event, len(r.events))
	copy(eventsCopy, r.events)
	return eventsCopy
}

func (r *StudentRepo) RequireStudentByID(t *testing.T, id user.ID) *user.StudentAssertions {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	student, exists := r.dbByID[id]
	if !exists {
		t.Fatalf("student with ID %s does not exist", id)
	}

	return user.NewStudentAssertions(student)
}

func (r *StudentRepo) RequireStudentByEmail(t *testing.T, email string) *user.StudentAssertions {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	student, exists := r.dbByEmail[email]
	if !exists {
		t.Fatalf("student with email %s does not exist", email)
	}

	return user.NewStudentAssertions(student)
}

func (r *StudentRepo) AssertStudentNotExistsByID(t *testing.T, id user.ID) *StudentRepo {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbByID[id]; exists {
		t.Errorf("expected student with ID %s to not exist, but it does", id)
	}
	return r
}

func (r *StudentRepo) AssertStudentNotExistsByEmail(t *testing.T, email string) *StudentRepo {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbByEmail[email]; exists {
		t.Errorf("expected student with email %s to not exist, but it does", email)
	}
	return r
}
