package user

import "github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"

type Staff struct {
	User
}

type RegisterStaffArgs struct {
	RegisterUserArgs
}

func RegisterStaff(p RegisterStaffArgs) *Staff {
	user, err := RegisterUser(p.RegisterUserArgs)
	if err != nil {
		return nil
	}
	user.role = role.StaffRole

	return &Staff{
		User: *user,
	}
}

type RehydrateStaffArgs struct {
	RehydrateUserArgs
}

func RehydrateStaff(p RehydrateStaffArgs) *Staff {
	return &Staff{
		User: *RehydrateUser(p.RehydrateUserArgs),
	}
}
