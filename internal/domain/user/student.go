package user

import (
	"errors"

	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
)

type Student struct {
	User

	major major.Major
	group string
	year  string
}

type RegisterStudentArgs struct {
	RegisterUserArgs
	Major major.Major
	Group string
	Year  string
}

func RegisterStudent(p RegisterStudentArgs) (*Student, error) {
	user, err := RegisterUser(p.RegisterUserArgs)
	if err != nil {
		return nil, err
	}
	if p.Major == "" || !major.IsValid(p.Major) {
		return nil, errors.New("invalid major")
	}
	if p.Group == "" {
		return nil, errors.New("group cannot be empty")
	}
	if p.Year == "" {
		return nil, errors.New("year cannot be empty")
	}

	return &Student{
		User:  *user,
		major: p.Major,
		group: p.Group,
		year:  p.Year,
	}, nil
}

type RehydrateStudentArgs struct {
	RehydrateUserArgs
	Major string
	Group string
	Year  string
}

func RehydrateStudent(p RehydrateStudentArgs) *Student {
	return &Student{
		User:  *RehydrateUser(p.RehydrateUserArgs),
		major: major.Major(p.Major),
		group: p.Group,
		year:  p.Year,
	}
}

func (s *Student) Major() major.Major {
	if s == nil {
		return ""
	}

	return s.major
}

func (s *Student) Group() string {
	if s == nil {
		return ""
	}

	return s.group
}

func (s *Student) Year() string {
	if s == nil {
		return ""
	}

	return s.year
}

func (s *Student) SetMajor(m major.Major) error {
	if s == nil {
		return errors.New("student is nil")
	}
	if !major.IsValid(m) {
		return errors.New("invalid major")
	}

	s.major = m
	return nil
}

func (s *Student) SetGroup(group string) error {
	if s == nil {
		return errors.New("student is nil")
	}
	if group == "" {
		return errors.New("group cannot be empty")
	}

	s.group = group
	return nil
}

func (s *Student) SetYear(year string) error {
	if s == nil {
		return errors.New("student is nil")
	}
	if year == "" {
		return errors.New("year cannot be empty")
	}

	s.year = year
	return nil
}
