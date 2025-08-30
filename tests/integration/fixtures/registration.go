package fixtures

import (
	"strings"

	"github.com/google/uuid"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
)

// Test emails
const (
	ValidStudentEmail  = "student@test.com"
	ValidStudent2Email = "student2@test.com"
	ValidStudent3Email = "student3@test.com"
	ValidStudent4Email = "student4@test.com"
	ValidStaffEmail    = "staff@test.com"
	ValidStaff2Email   = "staff2@test.com"
	ValidStaff3Email   = "staff3@test.com"
	ValidStaff4Email   = "staff4@test.com"
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
	TestStudentBarcode  = user.Barcode("210107")
	TestStudent2Barcode = user.Barcode("210108")
	TestStaffBarcode    = user.Barcode("230001")
	TestStaff2Barcode   = user.Barcode("230002")

	TestStudent = struct {
		ID        user.ID
		Barcode   user.Barcode
		Username  string
		Email     string
		FirstName string
		LastName  string
		Password  string
		GroupID   group.ID
		Major     major.Major
	}{
		ID:        user.ID(uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")),
		Barcode:   TestStudentBarcode,
		Username:  "teststudent",
		Email:     ValidStudentEmail,
		FirstName: "Test",
		LastName:  "Student",
		Password:  "SecurePass123!",
		GroupID:   SEGroup.ID,
		Major:     major.SE,
	}

	TestStudent2 = struct {
		ID        user.ID
		Barcode   user.Barcode
		Username  string
		Email     string
		FirstName string
		LastName  string
		Password  string
		GroupID   group.ID
		Major     major.Major
	}{
		ID:        user.ID(uuid.MustParse("990e8400-e29b-41d4-a716-446655440001")),
		Barcode:   TestStudent2Barcode,
		Username:  "teststudent2",
		Email:     ValidStudent2Email,
		FirstName: "Test",
		LastName:  "Student2",
		Password:  "AnotherPass123!",
		GroupID:   ITGroup.ID,
		Major:     major.IT,
	}

	TestStaff = struct {
		ID        user.ID
		Barcode   user.Barcode
		Username  string
		Email     string
		FirstName string
		LastName  string
		Password  string
	}{
		ID:        user.ID(uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")),
		Barcode:   TestStaffBarcode,
		Username:  "teststaff",
		Email:     ValidStaffEmail,
		FirstName: "Test",
		LastName:  "Staff",
		Password:  "StaffPass123!",
	}

	TestStaff2 = struct {
		ID        user.ID
		Barcode   user.Barcode
		Username  string
		Email     string
		FirstName string
		LastName  string
		Password  string
	}{
		ID:        user.ID(uuid.MustParse("880e8400-e29b-41d4-a716-446655440001")),
		Barcode:   TestStaff2Barcode,
		Username:  "teststaff2",
		Email:     ValidStaff2Email,
		FirstName: "Test2",
		LastName:  "Staff2",
		Password:  "StaffPass456!",
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
		ID    group.ID
		Name  string
		Year  string
		Major major.Major
	}{
		ID:    group.ID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")),
		Name:  "SE-2301",
		Year:  "2023",
		Major: major.SE,
	}

	ITGroup = struct {
		ID    group.ID
		Name  string
		Year  string
		Major major.Major
	}{
		ID:    group.ID(uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")),
		Name:  "CS-2301",
		Year:  "2023",
		Major: major.IT,
	}

	CSGroup = struct {
		ID    group.ID
		Name  string
		Year  string
		Major major.Major
	}{
		ID:    group.ID(uuid.MustParse("770e8400-e29b-41d4-a716-446655440002")),
		Name:  "CS-2301",
		Year:  "2023",
		Major: major.CS,
	}
)
