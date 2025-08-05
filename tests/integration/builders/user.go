package builders

import (
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/major"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
)

type UserBuilder struct {
	id        user.ID
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
	hash, _ := bcrypt.GenerateFromPassword([]byte("Test123!"), 10)
	now := time.Now()

	return &UserBuilder{
		id:        user.ID("test-user-id"),
		firstName: "Test",
		lastName:  "User",
		email:     "test@example.com",
		password:  "Test123!",
		passHash:  hash,
		avatarURL: "",
		role:      role.StudentRole,
		createdAt: now,
		updatedAt: now,
	}
}

func (b *UserBuilder) WithID(id string) *UserBuilder {
	b.id = user.ID(id)
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
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
	b.passHash = hash
	return b
}

func (b *UserBuilder) WithRole(role role.Global) *UserBuilder {
	b.role = role
	return b
}

func (b *UserBuilder) AsStudent() *UserBuilder {
	b.role = role.StudentRole
	return b
}

func (b *UserBuilder) AsStaff() *UserBuilder {
	b.role = role.StaffRole
	return b
}

func (b *UserBuilder) AsAITUSA() *UserBuilder {
	b.role = role.AITUSARole
	return b
}

func (b *UserBuilder) Build() *user.User {
	return user.RehydrateUser(user.RehydrateUserArgs{
		ID:        b.id,
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

func (b *UserBuilder) BuildNew() (*user.User, error) {
	return user.RegisterUser(user.RegisterUserArgs{
		ID:        b.id,
		FirstName: b.firstName,
		LastName:  b.lastName,
		AvatarURL: b.avatarURL,
		Email:     b.email,
		Password:  b.password,
	})
}

// StudentBuilder extends UserBuilder for student-specific properties
type StudentBuilder struct {
	UserBuilder
	major major.Major
	group string
	year  string
}

func NewStudentBuilder() *StudentBuilder {
	return &StudentBuilder{
		UserBuilder: *NewUserBuilder().AsStudent(),
		major:       major.SE,
		group:       "SE-2301",
		year:        "2023",
	}
}

func (b *StudentBuilder) WithMajor(m major.Major) *StudentBuilder {
	b.major = m
	return b
}

func (b *StudentBuilder) WithGroup(group string) *StudentBuilder {
	b.group = group
	return b
}

func (b *StudentBuilder) WithYear(year string) *StudentBuilder {
	b.year = year
	return b
}

func (b *StudentBuilder) Build() *user.Student {
	return user.RehydrateStudent(user.RehydrateStudentArgs{
		RehydrateUserArgs: user.RehydrateUserArgs{
			ID:        b.id,
			FirstName: b.firstName,
			LastName:  b.lastName,
			Role:      role.StudentRole,
			AvatarURL: b.avatarURL,
			Email:     b.email,
			PassHash:  b.passHash,
			CreatedAt: b.createdAt,
			UpdatedAt: b.updatedAt,
		},
		Major: string(b.major),
		Group: b.group,
		Year:  b.year,
	})
}

func (b *StudentBuilder) RehydrateStudentArgs() user.RehydrateStudentArgs {
	return user.RehydrateStudentArgs{
		RehydrateUserArgs: b.UserBuilder.RehydrateArgs(),
		Major:             string(b.major),
		Group:             b.group,
		Year:              b.year,
	}
}

func (b *StudentBuilder) BuildNew() (*user.Student, error) {
	return user.RegisterStudent(user.RegisterStudentArgs{
		RegisterUserArgs: user.RegisterUserArgs{
			ID:        b.id,
			FirstName: b.firstName,
			LastName:  b.lastName,
			AvatarURL: b.avatarURL,
			Email:     b.email,
			Password:  b.password,
		},
		Major: b.major,
		Group: b.group,
		Year:  b.year,
	})
}
