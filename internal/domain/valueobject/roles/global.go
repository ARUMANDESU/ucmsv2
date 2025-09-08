package roles

type Global string

const (
	Unknown = Global("unknown")
	Guest   = Global("guest")
	Student = Global("student")
	AITUSA  = Global("aitusa")
	Staff   = Global("staff")
)

func (g Global) String() string {
	return string(g)
}

func IsGlobalValid[T Global | string](role T) bool {
	switch Global(role) {
	case Guest, Student, AITUSA, Staff:
		return true
	default:
		return false
	}
}
