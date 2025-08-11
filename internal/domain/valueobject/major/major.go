package major

type Major string

const (
	Unknown = Major("")
	IT      = Major("Computer Science")
	SE      = Major("Software Engineering")
	MT      = Major("Media Technology")
	CS      = Major("Cyber Security")
	BDA     = Major("Big Data Analysis")
	BDH     = Major("Big Data in Health")
	ITM     = Major("IT Management")
	ITE     = Major("IT Enterpreneurship")
	EE      = Major("Electronic Engineering")
	IoT     = Major("Internet of Things")
	ST      = Major("Smart Technology")
	DJ      = Major("Digital Journalism")
	MCs     = Major("Master of Computer Science")
)

func (m Major) String() string {
	return string(m)
}

func IsValid[T Major | string](major T) bool {
	switch Major(major) {
	case IT, SE, MT, CS, BDA, BDH, ITM, ITE, EE, IoT, ST, DJ, MCs:
		return true
	default:
		return false
	}
}
