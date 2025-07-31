package role

type Global string

const (
	GuestRole   = Global("guest")
	StudentRole = Global("student")
	AITUSARole  = Global("aitusa")
	StaffRole   = Global("staff")
)

func IsGlobalValid[T Global | string](role T) bool {
	switch Global(role) {
	case GuestRole, StudentRole, AITUSARole, StaffRole:
		return true
	default:
		return false
	}
}
