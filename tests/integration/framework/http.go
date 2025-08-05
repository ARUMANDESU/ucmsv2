package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type HTTPHelper struct {
	handler chi.Router
}

type Request struct {
	Path    string
	Method  string
	Body    any
	Headers map[string]string
	Query   map[string]string
	Context context.Context
}

type Response struct {
	*httptest.ResponseRecorder
	t *testing.T
}

func (h *HTTPHelper) Do(t *testing.T, req Request) *Response {
	t.Helper()

	var body io.Reader
	if req.Body != nil {
		jsonbytes, err := json.Marshal(req.Body)
		require.NoError(t, err)
		body = bytes.NewReader(jsonbytes)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, body)

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	if body == nil && req.Headers["Content-Type"] == "" {
		req.Headers["Content-Type"] = "application/json"
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if req.Query != nil {
		q := httpReq.URL.Query()
		for k, v := range req.Query {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	if req.Context != nil {
		httpReq.WithContext(req.Context)
	}

	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, httpReq)

	return &Response{ResponseRecorder: w, t: t}
}

func (r *Response) AssertStatus(expected int) *Response {
	r.t.Helper()

	assert.Equal(r.t, expected, r.Result().StatusCode, "unexpected status code")
	return r
}

func (r *Response) AssertSuccess() *Response {
	r.t.Helper()
	r.AssertStatus(http.StatusOK)

	var resp map[string]any
	r.ParseJSON(resp)
	assert.True(r.t, resp["success"].(bool), "expected succeeded=true")

	return r
}

func (r *Response) ParseJSON(v any) *Response {
	r.t.Helper()

	err := json.Unmarshal(r.Body.Bytes(), v)
	require.NoError(r.t, err, "failed to parse JSON response")

	return r
}

func (r *Response) AssertHeader(key, value string) *Response {
	r.t.Helper()

	actual := r.Header().Get(key)
	require.Equal(r.t, value, actual, fmt.Sprintf("expected header %s=%s, got %s", key, value, actual))
	return r
}

type RequestBuilder struct {
	req Request
}

func NewRequest(method, path string) *RequestBuilder {
	return &RequestBuilder{
		req: Request{
			Path:    path,
			Method:  method,
			Body:    nil,
			Headers: make(map[string]string),
			Query:   make(map[string]string),
			Context: nil,
		},
	}
}

func (b *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	b.req.Context = ctx
	return b
}

func (b *RequestBuilder) WithJSON(body any) *RequestBuilder {
	b.req.Body = body
	return b
}

func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	if b.req.Headers == nil {
		b.req.Headers = make(map[string]string)
	}
	b.req.Headers[key] = value
	return b
}

func (b *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	if b.req.Query == nil {
		b.req.Query = make(map[string]string)
	}
	b.req.Query[key] = value
	return b
}

func (b *RequestBuilder) Build() Request {
	return b.req
}
