package http

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
)

type MultipartFormBuilder struct {
	writer *multipart.Writer
	buf    *bytes.Buffer
}

func NewMultipartFormBuilder() *MultipartFormBuilder {
	buf := &bytes.Buffer{}
	return &MultipartFormBuilder{
		writer: multipart.NewWriter(buf),
		buf:    buf,
	}
}

func (b *MultipartFormBuilder) AddFile(fieldName, fileName, contentType string, data []byte) *MultipartFormBuilder {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"%s\"; filename=\"%s\"", fieldName, fileName))
	h.Set("Content-Type", contentType)

	part, err := b.writer.CreatePart(h)
	if err != nil {
		panic(fmt.Sprintf("failed to create multipart part: %v", err))
	}

	if len(data) > 0 {
		_, err = part.Write(data)
		if err != nil {
			panic(fmt.Sprintf("failed to write data to multipart part: %v", err))
		}
	}

	return b
}

func (b *MultipartFormBuilder) AddField(fieldName, value string) *MultipartFormBuilder {
	err := b.writer.WriteField(fieldName, value)
	if err != nil {
		panic(fmt.Sprintf("failed to write field to multipart form: %v", err))
	}
	return b
}

func (b *MultipartFormBuilder) Build() (io.Reader, string) {
	err := b.writer.Close()
	if err != nil {
		panic(fmt.Sprintf("failed to close multipart writer: %v", err))
	}

	return b.buf, b.writer.FormDataContentType()
}
