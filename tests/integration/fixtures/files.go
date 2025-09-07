package fixtures

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"
)

const (
	MinAvatarSize = 100
	MaxAvatarSize = 5 * 1024 * 1024
)

var (
	ValidJPEGAvatar = createValidJPEG()
	ValidPNGAvatar  = createValidPNG()
	ValidGIFAvatar  = createValidGIF()
	ValidWebPAvatar = createValidWebP()

	TinyJPEGAvatar    = createTinyJPEG()
	LargeJPEGAvatar   = createLargeJPEG()
	MaxSizeJPEGAvatar = createMaxSizeJPEG()

	CorruptedJPEGAvatar = createCorruptedJPEG()
	InvalidFormatAvatar = createInvalidFormat()
	EmptyAvatar         = createEmpty()
)

type AvatarFile struct {
	Data        []byte
	ContentType string
	Size        int64
	IsValid     bool
	Description string
}

func (af *AvatarFile) Reader() io.Reader {
	return bytes.NewReader(af.Data)
}

func (af *AvatarFile) SizeBytes() int64 {
	return int64(len(af.Data))
}

var AvatarFileFixtures = map[string]*AvatarFile{
	"valid_jpeg": {
		Data:        ValidJPEGAvatar,
		ContentType: "image/jpeg",
		Size:        int64(len(ValidJPEGAvatar)),
		IsValid:     true,
		Description: "Valid JPEG avatar (1KB)",
	},
	"valid_png": {
		Data:        ValidPNGAvatar,
		ContentType: "image/png",
		Size:        int64(len(ValidPNGAvatar)),
		IsValid:     true,
		Description: "Valid PNG avatar (1KB)",
	},
	"valid_gif": {
		Data:        ValidGIFAvatar,
		ContentType: "image/gif",
		Size:        int64(len(ValidGIFAvatar)),
		IsValid:     true,
		Description: "Valid GIF avatar (1KB)",
	},
	"valid_webp": {
		Data:        ValidWebPAvatar,
		ContentType: "image/webp",
		Size:        int64(len(ValidWebPAvatar)),
		IsValid:     true,
		Description: "Valid WebP avatar (1KB)",
	},
	"tiny_jpeg": {
		Data:        TinyJPEGAvatar,
		ContentType: "image/jpeg",
		Size:        int64(len(TinyJPEGAvatar)),
		IsValid:     false,
		Description: "JPEG avatar too small (50 bytes)",
	},
	"large_jpeg": {
		Data:        LargeJPEGAvatar,
		ContentType: "image/jpeg",
		Size:        int64(len(LargeJPEGAvatar)),
		IsValid:     true,
		Description: "Large JPEG avatar (1MB)",
	},
	"max_size_jpeg": {
		Data:        MaxSizeJPEGAvatar,
		ContentType: "image/jpeg",
		Size:        int64(len(MaxSizeJPEGAvatar)),
		IsValid:     true,
		Description: "Maximum size JPEG avatar (5MB)",
	},
	"corrupted_jpeg": {
		Data:        CorruptedJPEGAvatar,
		ContentType: "image/jpeg",
		Size:        int64(len(CorruptedJPEGAvatar)),
		IsValid:     false,
		Description: "Corrupted JPEG data",
	},
	"invalid_format": {
		Data:        InvalidFormatAvatar,
		ContentType: "application/pdf",
		Size:        int64(len(InvalidFormatAvatar)),
		IsValid:     false,
		Description: "Invalid file format (PDF)",
	},
	"empty": {
		Data:        EmptyAvatar,
		ContentType: "image/jpeg",
		Size:        0,
		IsValid:     false,
		Description: "Empty file",
	},
}

func createValidJPEG() []byte {
	jpegHeader := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9Pjv/2wBDAQoLCw4NDhwQEBw7KCIoOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozv/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
	data, _ := base64.StdEncoding.DecodeString(jpegHeader)
	padding := make([]byte, 1024-len(data))
	return append(data, padding...)
}

func createValidPNG() []byte {
	pngHeader := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	data, _ := base64.StdEncoding.DecodeString(pngHeader)
	padding := make([]byte, 1024-len(data))
	return append(data, padding...)
}

func createValidGIF() []byte {
	gifHeader := "R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"
	data, _ := base64.StdEncoding.DecodeString(gifHeader)
	padding := make([]byte, 1024-len(data))
	return append(data, padding...)
}

func createValidWebP() []byte {
	webpHeader := "UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA"
	data, _ := base64.StdEncoding.DecodeString(webpHeader)
	padding := make([]byte, 1024-len(data))
	return append(data, padding...)
}

func createTinyJPEG() []byte {
	jpegHeader := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9Pjv/9k="
	data, _ := base64.StdEncoding.DecodeString(jpegHeader)
	return data[:50]
}

func createLargeJPEG() []byte {
	jpegHeader := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9Pjv/2wBDAQoLCw4NDhwQEBw7KCIoOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozv/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
	data, _ := base64.StdEncoding.DecodeString(jpegHeader)
	size := 1024 * 1024
	padding := make([]byte, size-len(data))
	return append(data, padding...)
}

func createMaxSizeJPEG() []byte {
	jpegHeader := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9Pjv/2wBDAQoLCw4NDhwQEBw7KCIoOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozv/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
	data, _ := base64.StdEncoding.DecodeString(jpegHeader)
	size := MaxAvatarSize
	padding := make([]byte, size-len(data))
	return append(data, padding...)
}

func createCorruptedJPEG() []byte {
	return []byte("This is not a valid JPEG file content, just random bytes that should fail image validation")
}

func createInvalidFormat() []byte {
	pdfHeader := "%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	padding := make([]byte, 1024-len(pdfHeader))
	return append([]byte(pdfHeader), padding...)
}

func createEmpty() []byte {
	return []byte{}
}

func CreateOversizedJPEG() []byte {
	jpegHeader := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9Pjv/2wBDAQoLCw4NDhwQEBw7KCIoOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozv/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
	data, _ := base64.StdEncoding.DecodeString(jpegHeader)
	size := MaxAvatarSize + 1024
	padding := make([]byte, size-len(data))
	return append(data, padding...)
}

func CreateRandomJPEGWithSize(targetSize int) []byte {
	jpegHeader := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9Pjv/2wBDAQoLCw4NDhwQEBw7KCIoOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozv/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
	data, _ := base64.StdEncoding.DecodeString(jpegHeader)

	if targetSize <= len(data) {
		return data[:targetSize]
	}

	padding := make([]byte, targetSize-len(data))
	for i := range padding {
		padding[i] = byte(i % 256)
	}
	return append(data, padding...)
}

func GetAvatarByKey(key string) *AvatarFile {
	return AvatarFileFixtures[key]
}

func GetValidAvatars() []*AvatarFile {
	var valid []*AvatarFile
	for _, fixture := range AvatarFileFixtures {
		if fixture.IsValid {
			valid = append(valid, fixture)
		}
	}
	return valid
}

func GetInvalidAvatars() []*AvatarFile {
	var invalid []*AvatarFile
	for _, fixture := range AvatarFileFixtures {
		if !fixture.IsValid {
			invalid = append(invalid, fixture)
		}
	}
	return invalid
}

func CreateMultipartFormData(filename, contentType string, fileData []byte) (io.Reader, string) {
	boundary := "----WebKitFormBoundary7MA4YWxkTrZu0gW"

	var body strings.Builder
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"avatar\"; filename=\"" + filename + "\"\r\n")
	body.WriteString("Content-Type: " + contentType + "\r\n")
	body.WriteString("\r\n")

	bodyBytes := []byte(body.String())
	bodyBytes = append(bodyBytes, fileData...)
	bodyBytes = append(bodyBytes, []byte("\r\n--"+boundary+"--\r\n")...)

	return bytes.NewReader(bodyBytes), "multipart/form-data; boundary=" + boundary
}
