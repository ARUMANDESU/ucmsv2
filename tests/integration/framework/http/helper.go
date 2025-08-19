package http

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

type Helper struct {
	handler chi.Router
}

func NewHelper(handler chi.Router) *Helper {
	return &Helper{handler: handler}
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

func (h *Helper) Do(t *testing.T, req Request) *Response {
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

	var resp map[string]any
	r.ParseJSON(&resp)
	message, ok := resp["message"].(string)
	if !ok {
		message = "no message in response"
	}

	assert.Equal(r.t, expected, r.Result().StatusCode, "unexpected status code, message: %s", message)
	return r
}

func (r *Response) AssertHeader(key, value string) *Response {
	r.t.Helper()

	actual := r.Header().Get(key)
	require.Equal(r.t, value, actual, fmt.Sprintf("expected header %s=%s, got %s", key, value, actual))
	return r
}

func (r *Response) AssertMessage(expected string) *Response {
	r.t.Helper()

	var resp map[string]any
	r.ParseJSON(&resp)
	message, ok := resp["message"].(string)
	require.True(r.t, ok, "expected message to be a string")
	assert.Equal(r.t, expected, message, "unexpected message in response")

	return r
}

func (r *Response) AssertContainsMessage(expected string) *Response {
	r.t.Helper()
	if expected == "" {
		return r
	}

	var resp map[string]any
	r.ParseJSON(&resp)
	message, ok := resp["message"].(string)
	require.True(r.t, ok, "expected message to be a string")
	assert.Contains(r.t, message, expected, "message does not contain expected text")

	return r
}

func (r *Response) AssertSuccess() *Response {
	r.t.Helper()
	r.AssertStatus(http.StatusOK)

	var resp map[string]any
	r.ParseJSON(&resp)
	assert.True(r.t, resp["success"].(bool), "expected succeeded=true")

	return r
}

func (r *Response) AssertAccepted() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusAccepted)
}

func (r *Response) AssertError(expectedStatus int, expectedMessage string) *Response {
	r.t.Helper()
	r.AssertStatus(expectedStatus)

	var resp map[string]any
	r.ParseJSON(&resp)
	require.False(r.t, resp["succeeded"].(bool), "expected succeeded=false")
	assert.Contains(r.t, resp["message"].(string), expectedMessage)
	return r
}

func (r *Response) AssertBadRequest() *Response {
	r.t.Helper()
	r.AssertStatus(http.StatusBadRequest)

	return r
}

func (r *Response) ParseJSON(v any) *Response {
	r.t.Helper()

	err := json.Unmarshal(r.Body.Bytes(), v)
	require.NoError(r.t, err, "failed to parse JSON response: %s", r.Body.String())

	return r
}

func (r *Response) GetCookie(name string) *http.Cookie {
	r.t.Helper()

	cookie := r.Result().Cookies()
	for _, c := range cookie {
		if c.Name == name {
			return c
		}
	}
	require.Fail(r.t, fmt.Sprintf("cookie %s not found", name))
	return nil
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

func (b *RequestBuilder) WithCookies(cookies map[string]string) *RequestBuilder {
	if b.req.Headers == nil {
		b.req.Headers = make(map[string]string)
	}
	for key, value := range cookies {
		b.req.Headers[http.CanonicalHeaderKey("Cookie")] += fmt.Sprintf("%s=%s; ", key, value)
	}
	return b
}

func (b *RequestBuilder) Build() Request {
	return b.req
}

func SameSiteModeToString(mode http.SameSite) string {
	switch mode {
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return "Unknown"
	}
}
