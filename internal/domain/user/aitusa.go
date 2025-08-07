package user

type AITUSA struct {
	student Student
}

type RehydrateAITUSAArgs struct {
	RehydrateStudentArgs
}

func RehydrateAITUSA(p RehydrateAITUSAArgs) *AITUSA {
	return &AITUSA{
		student: *RehydrateStudent(p.RehydrateStudentArgs),
	}
}

func (a *AITUSA) Student() *Student {
	if a == nil {
		return nil
	}

	return &a.student
}
