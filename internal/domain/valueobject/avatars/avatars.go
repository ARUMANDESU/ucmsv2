package avatars

import "fmt"

type Source int

const (
	SourceUnknown Source = iota
	SourceS3
	SourceExternal
)

func (s Source) String() string {
	switch s {
	case SourceS3:
		return "s3"
	case SourceExternal:
		return "external"
	default:
		return "unknown"
	}
}

func SourceFromString(str string) Source {
	switch str {
	case "s3":
		return SourceS3
	case "external":
		return SourceExternal
	default:
		return SourceUnknown
	}
}

type Avatar struct {
	Source   Source
	S3Key    string
	External string
}

func NewS3Avatar(s3Key string) Avatar {
	return Avatar{
		Source: SourceS3,
		S3Key:  s3Key,
	}
}

func NewExternalAvatar(url string) Avatar {
	return Avatar{
		Source:   SourceExternal,
		External: url,
	}
}

func (a Avatar) IsZero() bool {
	return a.Source == SourceUnknown && a.S3Key == "" && a.External == ""
}

func (a Avatar) GetURL(s3BaseURL string) string {
	switch a.Source {
	case SourceS3:
		return fmt.Sprintf("%s/%s", s3BaseURL, a.S3Key)
	case SourceExternal:
		return a.External
	default:
		return ""
	}
}
