package fixtures

import (
	"fmt"
	"time"
)

var (
	ValidAvatarURL      = "https://example-bucket.s3.amazonaws.com/avatars/user123/avatar.jpg"
	ValidAvatarS3Key    = "avatars/user123/avatar.jpg"
	ValidAvatarS3Bucket = "example-bucket"
	ValidS3BaseURL      = "https://example-bucket.s3.amazonaws.com"

	EmptyAvatarURL   = ""
	EmptyAvatarS3Key = ""

	LongAvatarURL = "https://example-bucket.s3.amazonaws.com/avatars/" +
		"very-long-user-id-that-exceeds-normal-expectations-and-creates-a-very-long-path/" +
		"very-long-filename-that-might-be-generated-by-some-system-or-user-input-" +
		"with-additional-metadata-and-timestamps-and-other-information-that-makes-" +
		"the-url-quite-long-but-still-within-reasonable-limits-for-modern-systems.jpg"

	LongAvatarS3Key = "avatars/" +
		"very-long-user-id-that-exceeds-normal-expectations-and-creates-a-very-long-path/" +
		"very-long-filename-that-might-be-generated-by-some-system-or-user-input-" +
		"with-additional-metadata-and-timestamps-and-other-information-that-makes-" +
		"the-url-quite-long-but-still-within-reasonable-limits-for-modern-systems.jpg"

	MaxLengthAvatarURL = buildMaxLengthURL()

	DefaultAvatarURL = "https://example-bucket.s3.amazonaws.com/defaults/default-avatar.png"
	DefaultS3Key     = "defaults/default-avatar.png"

	TestAvatarURLs = [...]string{
		"https://example-bucket.s3.amazonaws.com/avatars/student1/profile.jpg",
		"https://example-bucket.s3.amazonaws.com/avatars/student2/image.png",
		"https://example-bucket.s3.amazonaws.com/avatars/staff1/avatar.webp",
		"https://example-bucket.s3.amazonaws.com/avatars/admin1/photo.jpg",
	}

	TestS3Keys = [...]string{
		"avatars/student1/profile.jpg",
		"avatars/student2/image.png",
		"avatars/staff1/avatar.webp",
		"avatars/admin1/photo.jpg",
	}

	ExternalAvatarURLs = [...]string{
		"https://lh3.googleusercontent.com/a/ACg8ocJXYZ123",
		"https://platform-lookaside.fbsbx.com/platform/profilepic/?asid=456789",
		"https://pbs.twimg.com/profile_images/1234567890/avatar.jpg",
		"https://github.com/user123.png?size=200",
	}

	InvalidAvatarURLs = [...]string{
		"not-a-url",
		"ftp://example.com/avatar.jpg",
		"data:image/jpeg;base64,/9j/4AAQSkZJRgABA...",
		"javascript:alert('xss')",
		"",
	}

	MaliciousS3Keys = [...]string{
		"../../../etc/passwd",
		"avatars/../config/database.yml",
		"avatars/user1/../../../secrets.env",
		"avatars/\x00null-byte-injection",
		"avatars/extremely-long-key-" + string(make([]byte, 2000)),
	}
)

type AvatarFixture struct {
	Description string
	URL         string
	S3Key       string
	IsValid     bool
	Source      string
}

var AvatarFixtures = [...]AvatarFixture{
	{
		Description: "Standard JPEG avatar",
		URL:         "https://example-bucket.s3.amazonaws.com/avatars/user1/avatar.jpg",
		S3Key:       "avatars/user1/avatar.jpg",
		IsValid:     true,
		Source:      "s3",
	},
	{
		Description: "PNG avatar with timestamp",
		URL:         "https://example-bucket.s3.amazonaws.com/avatars/user2/avatar_20240101.png",
		S3Key:       "avatars/user2/avatar_20240101.png",
		IsValid:     true,
		Source:      "s3",
	},
	{
		Description: "WebP avatar in subfolder",
		URL:         "https://example-bucket.s3.amazonaws.com/avatars/2024/01/user3.webp",
		S3Key:       "avatars/2024/01/user3.webp",
		IsValid:     true,
		Source:      "s3",
	},
	{
		Description: "Google OAuth avatar",
		URL:         "https://lh3.googleusercontent.com/a/ACg8ocJXYZ123",
		S3Key:       "",
		IsValid:     true,
		Source:      "external",
	},
	{
		Description: "GitHub OAuth avatar",
		URL:         "https://github.com/user123.png?size=200",
		S3Key:       "",
		IsValid:     true,
		Source:      "external",
	},
	{
		Description: "Empty avatar (default)",
		URL:         "",
		S3Key:       "",
		IsValid:     true,
		Source:      "default",
	},
	{
		Description: "Invalid URL format",
		URL:         "not-a-valid-url",
		S3Key:       "",
		IsValid:     false,
		Source:      "invalid",
	},
	{
		Description: "Malicious path traversal",
		URL:         "",
		S3Key:       "../../../etc/passwd",
		IsValid:     false,
		Source:      "malicious",
	},
}

func GenerateAvatarS3Key(userID, extension string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("avatars/%s/avatar_%d.%s", userID, timestamp, extension)
}

func GenerateAvatarURL(s3BaseURL, s3Key string) string {
	if s3Key == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", s3BaseURL, s3Key)
}

func GetRandomAvatarS3Key() string {
	keys := TestS3Keys
	return keys[time.Now().UnixNano()%int64(len(keys))]
}

func GetRandomExternalAvatarURL() string {
	urls := ExternalAvatarURLs
	return urls[time.Now().UnixNano()%int64(len(urls))]
}

func buildMaxLengthURL() string {
	baseURL := "https://example-bucket.s3.amazonaws.com/avatars/"

	remaining := 1000 - len(baseURL) - len(".jpg")

	padding := string(make([]byte, remaining))
	for i := range padding {
		padding = string(rune('a' + (i % 26)))
	}

	return baseURL + padding + ".jpg"
}

type S3Config struct {
	Bucket    string
	Region    string
	BaseURL   string
	KeyPrefix string
}

var TestS3Config = S3Config{
	Bucket:    "ucms-test-avatars",
	Region:    "us-west-2",
	BaseURL:   "https://ucms-test-avatars.s3.us-west-2.amazonaws.com",
	KeyPrefix: "avatars/",
}

var ProductionS3Config = S3Config{
	Bucket:    "ucms-prod-avatars",
	Region:    "us-west-2",
	BaseURL:   "https://ucms-prod-avatars.s3.us-west-2.amazonaws.com",
	KeyPrefix: "avatars/",
}

func (c S3Config) BuildURL(s3Key string) string {
	if s3Key == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", c.BaseURL, s3Key)
}

func (c S3Config) BuildKey(userID, filename string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s%s/%d_%s", c.KeyPrefix, userID, timestamp, filename)
}
