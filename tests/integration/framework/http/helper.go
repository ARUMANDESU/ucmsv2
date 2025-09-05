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

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/roles"
	authhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/auth"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
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
		httpReq = httpReq.WithContext(req.Context)
	}

	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, httpReq)

	return &Response{ResponseRecorder: w, t: t}
}

func (r *Response) AssertStatus(expected int) *Response {
	r.t.Helper()

	assert.Equal(r.t, expected, r.Result().StatusCode,
		"unexpected status code: %d(%s), expected: %d(%s); message: %s; details: %s",
		r.Result().StatusCode,
		http.StatusText(r.Result().StatusCode),
		expected,
		http.StatusText(expected),
		r.getMessage(),
		r.getDetails(),
	)
	return r
}

func (r *Response) RequireStatus(expected int) *Response {
	r.t.Helper()

	require.Equal(r.t, expected, r.Result().StatusCode, "unexpected status code: %d(%s), expected: %d(%s); message: %s; details: %s",
		r.Result().StatusCode,
		http.StatusText(r.Result().StatusCode),
		expected,
		http.StatusText(expected),
		r.getMessage(),
		r.getDetails(),
	)
	return r
}

func (r *Response) AssertHeader(key, value string) *Response {
	r.t.Helper()

	actual := r.Header().Get(key)
	require.Equal(r.t, value, actual, fmt.Sprintf("expected header %s=%s, got %s", key, value, actual))
	return r
}

func (r *Response) AssertHeaderContains(key, value string) *Response {
	r.t.Helper()

	actual := r.Header().Get(key)
	require.Contains(r.t, actual, value, fmt.Sprintf("expected header %s to contain %s, got %s", key, value, actual))
	return r
}

func (r *Response) AssertMessage(expected string) *Response {
	r.t.Helper()

	var resp map[string]any
	r.RequireParseJSON(&resp)
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
	r.RequireParseJSON(&resp)
	message, ok := resp["message"].(string)
	require.True(r.t, ok, "expected message to be a string")
	assert.Contains(r.t, message, expected, "message does not contain expected text")

	return r
}

func (r *Response) AssertSuccess() *Response {
	r.t.Helper()
	r.AssertStatus(http.StatusOK)

	var resp map[string]any
	r.RequireParseJSON(&resp)
	assert.True(r.t, resp["success"].(bool), "expected succeeded=true")

	return r
}

func (r *Response) RequireSuccess() *Response {
	r.t.Helper()
	r.RequireStatus(http.StatusOK)

	var resp map[string]any
	r.RequireParseJSON(&resp)
	require.True(r.t, resp["success"].(bool), "expected succeeded=true")

	return r
}

func (r *Response) AssertAccepted() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusAccepted)
}

func (r *Response) RequireAccepted() *Response {
	r.t.Helper()
	return r.RequireStatus(http.StatusAccepted)
}

func (r *Response) AssertError(expectedStatus int, expectedMessage string) *Response {
	r.t.Helper()
	r.AssertStatus(expectedStatus)

	var resp map[string]any
	r.RequireParseJSON(&resp)
	require.False(r.t, resp["succeeded"].(bool), "expected succeeded=false")
	assert.Contains(r.t, resp["message"].(string), expectedMessage)
	return r
}

func (r *Response) AssertBadRequest() *Response {
	r.t.Helper()
	r.AssertStatus(http.StatusBadRequest)

	return r
}

func (r *Response) RequireParseJSON(v any) *Response {
	r.t.Helper()

	require.NotEmpty(r.t, r.Body, "response body is empty")

	err := json.Unmarshal(r.Body.Bytes(), v)
	require.NoError(r.t, err, "failed to parse JSON response: %s", r.Body.String())

	return r
}

func (r *Response) ParseJSONIfExists(v any) *Response {
	r.t.Helper()

	if r.Body.Len() == 0 {
		return r
	}

	err := json.Unmarshal(r.Body.Bytes(), v)
	if err != nil {
		return r
	}

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

func (r *Response) getMessage() string {
	var resp map[string]any
	r.ParseJSONIfExists(&resp)
	message, ok := resp["message"].(string)
	if !ok {
		return ""
	}
	return message
}

func (r *Response) getDetails() string {
	var resp map[string]any
	r.ParseJSONIfExists(&resp)
	details, ok := resp["details"].(string)
	if !ok {
		return ""
	}
	return details
}

type RequestBuilderOptions func(*RequestBuilder)

func WithStaff(t *testing.T, id user.ID) RequestBuilderOptions {
	token := builders.JWTFactory{}.
		AccessTokenBuilder(id.String(), roles.Staff.String()).
		BuildSignedStringT(t)
	return WithAccessTokenCookie(token)
}

func WithStudent(t *testing.T, id user.ID) RequestBuilderOptions {
	token := builders.JWTFactory{}.
		AccessTokenBuilder(id.String(), roles.Student.String()).
		BuildSignedStringT(t)
	return WithAccessTokenCookie(token)
}

// WithAccessTokenCookie adds access token cookie to the request to simulate authenticated user
func WithAccessTokenCookie(token string) RequestBuilderOptions {
	return func(b *RequestBuilder) {
		b.WithCookies([]string{
			(&http.Cookie{
				Name:     authhttp.AccessJWTCookie,
				Value:    token,
				Path:     "/",
				Domain:   "localhost",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}).String(),
		})
	}
}

// WithAnon removes access token cookie to simulate anonymous user
func WithAnon() RequestBuilderOptions {
	return func(b *RequestBuilder) {
		b.WithCookies([]string{
			(&http.Cookie{
				Name:     authhttp.AccessJWTCookie,
				Value:    "",
				Path:     "/",
				Domain:   "localhost",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   -1, // Delete the cookie
			}).String(),
		})
	}
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
	if b.req.Headers == nil {
		b.req.Headers = make(map[string]string)
	}
	b.req.Headers["Content-Type"] = "application/json"
	return b
}

// WithHeader adds a header to the request.
// Be aware WithJSON will automatically set the Content-Type header to application/json,
// so call WithHeader after WithJSON if you need change Content-Type header
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

func (b *RequestBuilder) WithCookies(cookies []string) *RequestBuilder {
	if b.req.Headers == nil {
		b.req.Headers = make(map[string]string)
	}
	for _, value := range cookies {
		b.req.Headers[http.CanonicalHeaderKey("Cookie")] += value + "; "
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
