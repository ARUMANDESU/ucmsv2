package builders

import (
	"fmt"
	"math/rand/v2"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/group"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
)

const TestPasswordCost = 4

type UserFactory struct{}

func (f *UserFactory) Student(email string) *user.Student {
	b := NewStudentBuilder()
	b.WithEmail(email)
	return b.Build()
}

func (f *UserFactory) Staff(email string) *user.Staff {
	return user.RehydrateStaff(user.RehydrateStaffArgs{
		RehydrateUserArgs: NewUserBuilder().
			WithEmail(email).
			AsStaff().
			RehydrateArgs(),
	})
}

func (f *UserFactory) AITUSA(email string) *user.AITUSA {
	student := NewStudentBuilder()
	student.WithEmail(email)

	return user.RehydrateAITUSA(user.RehydrateAITUSAArgs{
		RehydrateStudentArgs: student.RehydrateStudentArgs(),
	})
}

type UserBuilder struct {
	id        user.ID
	barcode   user.Barcode
	username  string
	firstName string
	lastName  string
	email     string
	password  string
	passHash  []byte
	avatarURL string
	role      role.Global
	createdAt time.Time
	updatedAt time.Time
}

func NewUserBuilder() *UserBuilder {
	hash, _ := bcrypt.GenerateFromPassword([]byte(fixtures.TestStudent.Password), user.PasswordCostFactor)
	now := time.Now()

	return &UserBuilder{
		id:        user.NewID(),
		barcode:   fixtures.TestStudent.Barcode,
		username:  fmt.Sprintf("user_%d_%d", rand.Uint()%1000, now.UnixNano()),
		firstName: fixtures.TestStudent.FirstName,
		lastName:  fixtures.TestStudent.LastName,
		email:     fixtures.ValidStudentEmail,
		password:  fixtures.TestStudent.Password,
		passHash:  hash,
		avatarURL: "",
		role:      role.Student,
		createdAt: now,
		updatedAt: now,
	}
}

func (b *UserBuilder) WithID(id user.ID) *UserBuilder {
	b.id = id
	return b
}

func (b *UserBuilder) WithBarcode(barcode user.Barcode) *UserBuilder {
	b.barcode = barcode
	return b
}

func (b *UserBuilder) WithUsername(username string) *UserBuilder {
	b.username = username
	return b
}

func (b *UserBuilder) WithRandomUsername() *UserBuilder {
	b.username = fmt.Sprintf("user_%d", time.Now().UnixNano())
	return b
}

func (b *UserBuilder) WithName(firstName, lastName string) *UserBuilder {
	b.firstName = firstName
	b.lastName = lastName
	return b
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.email = email
	return b
}

func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	b.password = password

	var err error
	if env.Current() == env.Test {
		b.passHash, err = bcrypt.GenerateFromPassword([]byte(password), 4)
	} else {
		b.passHash, err = bcrypt.GenerateFromPassword([]byte(password), user.PasswordCostFactor)
	}
	if err != nil {
		panic("failed to generate password hash: " + err.Error())
	}

	return b
}

func (b *UserBuilder) withPassHash(passHash []byte) *UserBuilder {
	b.passHash = passHash
	return b
}

func (b *UserBuilder) WithRole(role role.Global) *UserBuilder {
	b.role = role
	return b
}

func (b *UserBuilder) AsStudent() *UserBuilder {
	b.role = role.Student
	return b
}

func (b *UserBuilder) AsStaff() *UserBuilder {
	b.role = role.Staff
	return b
}

func (b *UserBuilder) AsAITUSA() *UserBuilder {
	b.role = role.AITUSA
	return b
}

func (b *UserBuilder) Build() *user.User {
	return user.RehydrateUser(user.RehydrateUserArgs{
		ID:        b.id,
		Barcode:   b.barcode,
		Username:  b.username,
		FirstName: b.firstName,
		LastName:  b.lastName,
		Role:      b.role,
		AvatarURL: b.avatarURL,
		Email:     b.email,
		PassHash:  b.passHash,
		CreatedAt: b.createdAt,
		UpdatedAt: b.updatedAt,
	})
}

func (b *UserBuilder) RehydrateArgs() user.RehydrateUserArgs {
	return user.RehydrateUserArgs{
		ID:        b.id,
		Barcode:   b.barcode,
		FirstName: b.firstName,
		LastName:  b.lastName,
		Role:      b.role,
		AvatarURL: b.avatarURL,
		Email:     b.email,
		PassHash:  b.passHash,
		CreatedAt: b.createdAt,
		UpdatedAt: b.updatedAt,
	}
}

func (b *UserBuilder) BuildNew() *user.User {
	return user.RehydrateUser(user.RehydrateUserArgs{
		ID:        b.id,
		Barcode:   b.barcode,
		FirstName: b.firstName,
		LastName:  b.lastName,
		AvatarURL: b.avatarURL,
		Email:     b.email,
		PassHash:  b.passHash,
		CreatedAt: b.createdAt,
		UpdatedAt: b.updatedAt,
		Role:      b.role,
	})
}

// StudentBuilder extends UserBuilder for student-specific properties
type StudentBuilder struct {
	UserBuilder
	groupID        group.ID
	registrationID registration.ID
}

func NewStudentBuilder() *StudentBuilder {
	return &StudentBuilder{
		UserBuilder:    *NewUserBuilder().AsStudent(),
		groupID:        fixtures.SEGroup.ID,
		registrationID: registration.ID(fixtures.ValidStudentRegistrationID),
	}
}

func (b *StudentBuilder) WithGroupID(groupID group.ID) *StudentBuilder {
	b.groupID = groupID
	return b
}

func (b *StudentBuilder) WithID(id user.ID) *StudentBuilder {
	b.UserBuilder.WithID(id)
	return b
}

// Override UserBuilder methods to return *StudentBuilder for proper chaining
func (b *StudentBuilder) WithBarcode(barcode user.Barcode) *StudentBuilder {
	b.UserBuilder.WithBarcode(barcode)
	return b
}

func (b *StudentBuilder) WithUsername(username string) *StudentBuilder {
	b.UserBuilder.WithUsername(username)
	return b
}

func (b *StudentBuilder) WithRegistrationID(registrationID registration.ID) *StudentBuilder {
	b.registrationID = registrationID
	return b
}

func (b *StudentBuilder) WithName(firstName, lastName string) *StudentBuilder {
	b.UserBuilder.WithName(firstName, lastName)
	return b
}

func (b *StudentBuilder) WithFirstName(firstName string) *StudentBuilder {
	b.UserBuilder.firstName = firstName
	return b
}

func (b *StudentBuilder) WithLastName(lastName string) *StudentBuilder {
	b.UserBuilder.lastName = lastName
	return b
}

func (b *StudentBuilder) WithEmail(email string) *StudentBuilder {
	b.UserBuilder.WithEmail(email)
	return b
}

func (b *StudentBuilder) WithPassword(password string) *StudentBuilder {
	b.UserBuilder.WithPassword(password)
	return b
}

func (b *StudentBuilder) WithPassHash(passHash []byte) *StudentBuilder {
	b.UserBuilder.withPassHash(passHash)
	return b
}

func (b *StudentBuilder) WithRole(role role.Global) *StudentBuilder {
	b.UserBuilder.WithRole(role)
	return b
}

func (b *StudentBuilder) AsStudent() *StudentBuilder {
	b.UserBuilder.AsStudent()
	return b
}

func (b *StudentBuilder) AsStaff() *StudentBuilder {
	b.UserBuilder.AsStaff()
	return b
}

func (b *StudentBuilder) AsAITUSA() *StudentBuilder {
	b.UserBuilder.AsAITUSA()
	return b
}

func (b *StudentBuilder) WithInvalidLongFirstName() *StudentBuilder {
	b.UserBuilder.firstName = fixtures.InvalidLongFirstName
	return b
}

func (b *StudentBuilder) WithInvalidShortFirstName() *StudentBuilder {
	b.UserBuilder.firstName = fixtures.InvalidShortFirstName
	return b
}

func (b *StudentBuilder) WithInvalidLongLastName() *StudentBuilder {
	b.UserBuilder.lastName = fixtures.InvalidLongLastName
	return b
}

func (b *StudentBuilder) WithInvalidShortLastName() *StudentBuilder {
	b.UserBuilder.lastName = fixtures.InvalidShortLastName
	return b
}

func (b *StudentBuilder) Build() *user.Student {
	return user.RehydrateStudent(user.RehydrateStudentArgs{
		RehydrateUserArgs: user.RehydrateUserArgs{
			ID:        b.id,
			Barcode:   b.barcode,
			Username:  b.username,
			FirstName: b.firstName,
			LastName:  b.lastName,
			Role:      role.Student,
			AvatarURL: b.avatarURL,
			Email:     b.email,
			PassHash:  b.passHash,
			CreatedAt: b.createdAt,
			UpdatedAt: b.updatedAt,
		},
		GroupID: b.groupID,
	})
}

func (b *StudentBuilder) RehydrateStudentArgs() user.RehydrateStudentArgs {
	return user.RehydrateStudentArgs{
		RehydrateUserArgs: b.UserBuilder.RehydrateArgs(),
		GroupID:           b.groupID,
	}
}

func (b *StudentBuilder) BuildNew() (*user.Student, error) {
	return user.RegisterStudent(user.RegisterStudentArgs{
		Barcode:        b.barcode,
		Username:       b.username,
		RegistrationID: b.registrationID,
		FirstName:      b.firstName,
		LastName:       b.lastName,
		AvatarURL:      b.avatarURL,
		Email:          b.email,
		Password:       b.password,
		GroupID:        b.groupID,
	})
}

func (b *StudentBuilder) BuildRegisterArgs() user.RegisterStudentArgs {
	return user.RegisterStudentArgs{
		Barcode:        b.barcode,
		Username:       b.username,
		RegistrationID: b.registrationID,
		FirstName:      b.firstName,
		LastName:       b.lastName,
		AvatarURL:      b.avatarURL,
		Email:          b.email,
		Password:       b.password,
		GroupID:        b.groupID,
	}
}

// StaffBuilder extends UserBuilder for staff-specific properties
type StaffBuilder struct {
	UserBuilder
	registrationID registration.ID
}

func NewStaffBuilder() *StaffBuilder {
	return &StaffBuilder{
		UserBuilder:    *NewUserBuilder().AsStaff(),
		registrationID: registration.ID(fixtures.ValidStaffRegistrationID),
	}
}

func (b *StaffBuilder) WithID(id user.ID) *StaffBuilder {
	b.UserBuilder.WithID(id)
	return b
}

// Override UserBuilder methods to return *StaffBuilder for proper chaining
func (b *StaffBuilder) WithBarcode(barcode user.Barcode) *StaffBuilder {
	b.UserBuilder.WithBarcode(barcode)
	return b
}

func (b *StaffBuilder) WithUsername(username string) *StaffBuilder {
	b.UserBuilder.WithUsername(username)
	return b
}

func (b *StaffBuilder) WithRegistrationID(registrationID registration.ID) *StaffBuilder {
	b.registrationID = registrationID
	return b
}

func (b *StaffBuilder) WithName(firstName, lastName string) *StaffBuilder {
	b.UserBuilder.WithName(firstName, lastName)
	return b
}

func (b *StaffBuilder) WithFirstName(firstName string) *StaffBuilder {
	b.UserBuilder.firstName = firstName
	return b
}

func (b *StaffBuilder) WithLastName(lastName string) *StaffBuilder {
	b.UserBuilder.lastName = lastName
	return b
}

func (b *StaffBuilder) WithEmail(email string) *StaffBuilder {
	b.UserBuilder.WithEmail(email)
	return b
}

func (b *StaffBuilder) WithPassword(password string) *StaffBuilder {
	b.UserBuilder.WithPassword(password)
	return b
}

func (b *StaffBuilder) WithPassHash(passHash []byte) *StaffBuilder {
	b.UserBuilder.withPassHash(passHash)
	return b
}

func (b *StaffBuilder) WithRole(role role.Global) *StaffBuilder {
	b.UserBuilder.WithRole(role)
	return b
}

func (b *StaffBuilder) AsStudent() *StaffBuilder {
	b.UserBuilder.AsStudent()
	return b
}

func (b *StaffBuilder) AsStaff() *StaffBuilder {
	b.UserBuilder.AsStaff()
	return b
}

func (b *StaffBuilder) AsAITUSA() *StaffBuilder {
	b.UserBuilder.AsAITUSA()
	return b
}

func (b *StaffBuilder) WithInvalidLongFirstName() *StaffBuilder {
	b.UserBuilder.firstName = fixtures.InvalidLongFirstName
	return b
}

func (b *StaffBuilder) WithInvalidShortFirstName() *StaffBuilder {
	b.UserBuilder.firstName = fixtures.InvalidShortFirstName
	return b
}

func (b *StaffBuilder) WithInvalidLongLastName() *StaffBuilder {
	b.UserBuilder.lastName = fixtures.InvalidLongLastName
	return b
}

func (b *StaffBuilder) WithInvalidShortLastName() *StaffBuilder {
	b.UserBuilder.lastName = fixtures.InvalidShortLastName
	return b
}

func (b *StaffBuilder) Build() *user.Staff {
	return user.RehydrateStaff(user.RehydrateStaffArgs{
		RehydrateUserArgs: user.RehydrateUserArgs{
			ID:        b.id,
			Barcode:   b.barcode,
			Username:  b.username,
			FirstName: b.firstName,
			LastName:  b.lastName,
			Role:      role.Staff,
			AvatarURL: b.avatarURL,
			Email:     b.email,
			PassHash:  b.passHash,
			CreatedAt: b.createdAt,
			UpdatedAt: b.updatedAt,
		},
	})
}

func (b *StaffBuilder) RehydrateStaffArgs() user.RehydrateStaffArgs {
	return user.RehydrateStaffArgs{
		RehydrateUserArgs: b.UserBuilder.RehydrateArgs(),
	}
}

func (b *StaffBuilder) BuildNew() (*user.Staff, error) {
	return user.RegisterStaff(user.RegisterStaffArgs{
		Barcode:        b.barcode,
		Username:       b.username,
		RegistrationID: b.registrationID,
		FirstName:      b.firstName,
		LastName:       b.lastName,
		AvatarURL:      b.avatarURL,
		Email:          b.email,
		Password:       b.password,
	})
}

func (b *StaffBuilder) BuildRegisterArgs() user.RegisterStaffArgs {
	return user.RegisterStaffArgs{
		Barcode:        b.barcode,
		Username:       b.username,
		RegistrationID: b.registrationID,
		FirstName:      b.firstName,
		LastName:       b.lastName,
		AvatarURL:      b.avatarURL,
		Email:          b.email,
		Password:       b.password,
	}
}
