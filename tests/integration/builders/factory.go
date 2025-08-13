package builders

type Factory struct {
	Registration *RegistrationFactory
	User         *UserFactory
	Group        *GroupFactory
}

func NewFactory() *Factory {
	return &Factory{
		Registration: &RegistrationFactory{},
		User:         &UserFactory{},
		Group:        &GroupFactory{},
	}
}
