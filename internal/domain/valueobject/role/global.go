package role

type Global string

const (
	Guest   = Global("guest")
	Student = Global("student")
	AITUSA  = Global("aitusa")
	Staff   = Global("staff")
)

func IsGlobalValid[T Global | string](role T) bool {
	switch Global(role) {
	case Guest, Student, AITUSA, Staff:
		return true
	default:
		return false
	}
}
