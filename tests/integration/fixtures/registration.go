package fixtures

import (
	"strings"

	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
)

// Test emails
const (
	ValidStudentEmail  = "student@astanait.edu.kz"
	ValidStudent2Email = "student2@astanait.edu.kz"
	ValidStaffEmail    = "staff@astanait.edu.kz"
	ValidExternalEmail = "external@gmail.com"
	InvalidEmail       = "notanemail"
)

var (
	ValidStudentRegistrationID  = uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	ValidStudent2RegistrationID = uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	ValidStaffRegistrationID    = uuid.MustParse("770e8400-e29b-41d4-a716-446655440001")
	ValidStaff2RegistrationID   = uuid.MustParse("770e8400-e29b-41d4-a716-446655440002")
)

var (
	InvalidLongFirstName  = strings.Repeat("A", user.MaxFirstNameLen+1)
	InvalidLongLastName   = strings.Repeat("B", user.MaxLastNameLen+1)
	InvalidShortFirstName = strings.Repeat("C", user.MinFirstNameLen-1)
	InvalidShortLastName  = strings.Repeat("D", user.MinLastNameLen-1)
)

// Test users
var (
	TestStudentBarcode  = "210107"
	TestStudent2Barcode = "210108"
	TestStaffBarcode    = "STAFF001"
	TestStaff2Barcode   = "STAFF002"

	TestStudent = struct {
		Barcode   string
		Email     string
		FirstName string
		LastName  string
		Password  string
		GroupID   uuid.UUID
		Major     major.Major
	}{
		Barcode:   TestStudentBarcode,
		Email:     ValidStudentEmail,
		FirstName: "Test",
		LastName:  "Student",
		Password:  "SecurePass123!",
		GroupID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Major:     major.SE,
	}

	TestStudent2 = struct {
		Barcode   string
		Email     string
		FirstName string
		LastName  string
		Password  string
		GroupID   uuid.UUID
		Major     major.Major
	}{
		Barcode:   TestStudent2Barcode,
		Email:     ValidStudent2Email,
		FirstName: "Test",
		LastName:  "Student2",
		Password:  "AnotherPass123!",
		GroupID:   uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
		Major:     major.IT,
	}

	TestStaff = struct {
		Barcode   string
		Email     string
		FirstName string
		LastName  string
		Password  string
	}{
		Barcode:   TestStaffBarcode,
		Email:     ValidStaffEmail,
		FirstName: "Test",
		LastName:  "Staff",
		Password:  "StaffPass123!",
	}
)

// Test verification codes
const (
	ValidVerificationCode   = "ABC123"
	InvalidVerificationCode = "WRONG1"
)

// Test groups
var (
	SEGroup = struct {
		ID    uuid.UUID
		Name  string
		Year  string
		Major major.Major
	}{
		ID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:  "SE-2301",
		Year:  "2023",
		Major: major.SE,
	}

	ITGroup = struct {
		ID    uuid.UUID
		Name  string
		Year  string
		Major major.Major
	}{
		ID:    uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
		Name:  "CS-2301",
		Year:  "2023",
		Major: major.IT,
	}

	CSGroup = struct {
		ID    uuid.UUID
		Name  string
		Year  string
		Major major.Major
	}{
		ID:    uuid.MustParse("770e8400-e29b-41d4-a716-446655440002"),
		Name:  "CS-2301",
		Year:  "2023",
		Major: major.CS,
	}
)
