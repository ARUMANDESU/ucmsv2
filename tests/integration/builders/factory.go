package builders

import "github.com/ARUMANDESU/ucms/internal/domain/user"

type Factory struct {
	Registration *RegistrationFactory
	User         *UserFactory
}

func NewFactory() *Factory {
	return &Factory{
		Registration: &RegistrationFactory{},
		User:         &UserFactory{},
	}
}

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
