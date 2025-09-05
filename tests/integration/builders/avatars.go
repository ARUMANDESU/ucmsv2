package builders

import (
	"fmt"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/avatars"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
)

// New AvatarBuilder for more complex avatar scenarios
type AvatarBuilder struct {
	source      string
	s3Key       string
	externalURL string
	s3Config    fixtures.S3Config
}

func NewAvatarBuilder() *AvatarBuilder {
	return &AvatarBuilder{
		source:   "default",
		s3Config: fixtures.TestS3Config,
	}
}

func (ab *AvatarBuilder) AsS3Avatar(userID, extension string) *AvatarBuilder {
	ab.source = "s3"
	ab.s3Key = fixtures.GenerateAvatarS3Key(userID, extension)
	ab.externalURL = ""
	return ab
}

func (ab *AvatarBuilder) AsExternalAvatar(url string) *AvatarBuilder {
	ab.source = "external"
	ab.s3Key = ""
	ab.externalURL = url
	return ab
}

func (ab *AvatarBuilder) AsGoogleAvatar() *AvatarBuilder {
	return ab.AsExternalAvatar("https://lh3.googleusercontent.com/a/ACg8ocJXYZ123")
}

func (ab *AvatarBuilder) AsGitHubAvatar(username string) *AvatarBuilder {
	return ab.AsExternalAvatar(fmt.Sprintf("https://github.com/%s.png?size=200", username))
}

func (ab *AvatarBuilder) AsDefaultAvatar() *AvatarBuilder {
	ab.source = "default"
	ab.s3Key = ""
	ab.externalURL = ""
	return ab
}

func (ab *AvatarBuilder) WithS3Config(config fixtures.S3Config) *AvatarBuilder {
	ab.s3Config = config
	return ab
}

func (ab *AvatarBuilder) Build() avatars.Avatar {
	return avatars.Avatar{
		Source:   avatars.SourceFromString(ab.source),
		S3Key:    ab.s3Key,
		External: ab.externalURL,
	}
}

func (ab *AvatarBuilder) URL() string {
	switch ab.source {
	case "s3":
		return ab.s3Config.BuildURL(ab.s3Key)
	case "external":
		return ab.externalURL
	default:
		return ""
	}
}

func (ab *AvatarBuilder) S3Key() string {
	return ab.s3Key
}

func (ab *AvatarBuilder) Source() string {
	return ab.source
}

// Test scenario helpers
func UserWithValidAvatar() *UserBuilder {
	return NewUserBuilder().WithS3Avatar(fixtures.ValidAvatarS3Key)
}

func UserWithExternalAvatar() *UserBuilder {
	return NewUserBuilder().WithExternalAvatar()
}

func UserWithoutAvatar() *UserBuilder {
	return NewUserBuilder().WithEmptyAvatar()
}
