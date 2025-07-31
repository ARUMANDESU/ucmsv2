package user

type AITUSA struct {
	Student
}

type RehydrateAITUSAArgs struct {
	RehydrateStudentArgs
}

func RehydrateAITUSA(p RehydrateAITUSAArgs) *AITUSA {
	return &AITUSA{
		Student: *RehydrateStudent(p.RehydrateStudentArgs),
	}
}
