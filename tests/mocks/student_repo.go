package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type StudentRepo struct {
	*EventRepo
	dbByEmail map[string]*user.Student
	dbByID    map[user.Barcode]*user.Student
	mu        sync.Mutex
}

func NewStudentRepo() *StudentRepo {
	return &StudentRepo{
		EventRepo: NewEventRepo(),
		dbByEmail: make(map[string]*user.Student),
		dbByID:    make(map[user.Barcode]*user.Student),
		mu:        sync.Mutex{},
	}
}

func (r *StudentRepo) GetStudentByEmail(ctx context.Context, email string) (*user.Student, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if student, exists := r.dbByEmail[email]; exists {
		return student, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *StudentRepo) GetStudentByBarcode(ctx context.Context, barcode user.Barcode) (*user.Student, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if student, exists := r.dbByID[barcode]; exists {
		return student, nil
	}
	return nil, errorx.NewNotFound()
}

func (r *StudentRepo) SaveStudent(ctx context.Context, student *user.Student) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if student == nil {
		return errors.New("student cannot be nil")
	}

	if _, exists := r.dbByEmail[student.User().Email()]; exists {
		return errorx.NewDuplicateEntry()
	}

	if _, exists := r.dbByID[student.User().Barcode()]; exists {
		return errorx.NewDuplicateEntry()
	}

	r.dbByEmail[student.User().Email()] = student
	r.dbByID[student.User().Barcode()] = student

	r.appendEvents(student.GetUncommittedEvents()...)

	return nil
}

func (r *StudentRepo) SeedStudent(t *testing.T, student *user.Student) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbByID[student.User().Barcode()]; exists {
		t.Fatalf("student with barcode %s already exists", student.User().Barcode())
	}

	if _, exists := r.dbByEmail[student.User().Email()]; exists {
		t.Fatalf("student with email %s already exists", student.User().Email())
	}

	r.dbByID[student.User().Barcode()] = student
	r.dbByEmail[student.User().Email()] = student
	r.appendEvents(student.GetUncommittedEvents()...)
}

func (r *StudentRepo) RequireStudentByBarcode(t *testing.T, barcode user.Barcode) *user.StudentAssertions {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	student, exists := r.dbByID[barcode]
	if !exists {
		t.Fatalf("student with barcode %s does not exist", barcode)
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

func (r *StudentRepo) AssertStudentNotExistsByBarcode(t *testing.T, barcode user.Barcode) *StudentRepo {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dbByID[barcode]; exists {
		t.Errorf("expected student with barcode %s to not exist, but it does", barcode)
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
