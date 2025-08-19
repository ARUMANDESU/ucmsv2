package builders

import (
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
)

type JWTFactory struct{}

func (f JWTFactory) AccessTokenBuilder(userID, userRole string) *JWTBuilder {
	return NewJWTBuilder().
		WithCookieName("ucmsv2_access").
		WithCookiePath("/").
		WithCookieDomain("localhost").
		WithIssuer("ucmsv2_auth").
		WithSubject("user").
		WithIssuedAt(time.Now()).
		WithExpiration(time.Now().Add(authapp.AccessTokenExpDuration)).
		WithDuration(authapp.AccessTokenExpDuration).
		WithUserID(userID).
		WithUserRole(userRole).
		WithSecret([]byte(fixtures.AccessTokenSecretKey)).
		WithSigningMethod(jwt.SigningMethodHS256)
}

func (f JWTFactory) RefreshTokenBuilder(userID string) *JWTBuilder {
	return NewJWTBuilder().
		WithCookieName("ucmsv2_refresh").
		WithCookiePath("/v1/auth/refresh").
		WithCookieDomain("localhost").
		WithIssuer("ucmsv2_auth").
		WithSubject("refresh").
		WithIssuedAt(time.Now()).
		WithExpiration(time.Now().Add(authapp.RefreshTokenExpDuration)).
		WithDuration(authapp.RefreshTokenExpDuration).
		WithUserID(userID).
		WithJTI(uuid.New().String()).
		WithClaim("scope", "refresh").
		WithSecret([]byte(fixtures.RefreshTokenSecretKey)).
		WithSigningMethod(jwt.SigningMethodHS256)
}

type JWTBuilder struct {
	secretKey     []byte
	signingMethod jwt.SigningMethod
	mapClaims     jwt.MapClaims
	tokenDuration *jwt.NumericDate
	cookieName    string
	cookiePath    string
	cookieDomain  string
}

func NewJWTBuilder() *JWTBuilder {
	return &JWTBuilder{
		secretKey:     []byte(fixtures.AccessTokenSecretKey),
		signingMethod: jwt.SigningMethodHS256,
		mapClaims:     jwt.MapClaims{},
		tokenDuration: jwt.NewNumericDate(time.Now().Add(authapp.AccessTokenExpDuration)),
	}
}

func (j *JWTBuilder) WithDuration(duration time.Duration) *JWTBuilder {
	j.tokenDuration = jwt.NewNumericDate(time.Now().Add(duration))
	j.mapClaims["exp"] = j.tokenDuration
	return j
}

func (j *JWTBuilder) WithIssuer(issuer string) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims["iss"] = issuer
	return j
}

func (j *JWTBuilder) WithSubject(subject string) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims["sub"] = subject
	return j
}

func (j *JWTBuilder) WithIssuedAt(issuedAt time.Time) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims["iat"] = jwt.NewNumericDate(issuedAt)
	return j
}

func (j *JWTBuilder) WithExpiration(expiration time.Time) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims["exp"] = jwt.NewNumericDate(expiration)
	return j
}

func (j *JWTBuilder) WithUserID(userID string) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims["uid"] = userID
	return j
}

func (j *JWTBuilder) WithUserRole(role string) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims["user_role"] = role
	return j
}

func (j *JWTBuilder) WithJTI(jti string) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims["jti"] = jti
	return j
}

func (j *JWTBuilder) WithSecret(key []byte) *JWTBuilder {
	j.secretKey = key
	return j
}

func (j *JWTBuilder) WithSigningMethod(method jwt.SigningMethod) *JWTBuilder {
	j.signingMethod = method
	return j
}

func (j *JWTBuilder) WithClaim(key string, value any) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	j.mapClaims[key] = value
	return j
}

func (j *JWTBuilder) WithClaimEmpty(key string) *JWTBuilder {
	if j.mapClaims == nil {
		j.mapClaims = make(jwt.MapClaims)
	}
	delete(j.mapClaims, key)
	return j
}

func (j *JWTBuilder) WithClaims(mapClaims jwt.MapClaims) *JWTBuilder {
	maps.Copy(j.mapClaims, mapClaims)
	return j
}

func (j *JWTBuilder) WithEmptyClaims() *JWTBuilder {
	j.mapClaims = jwt.MapClaims{}
	return j
}

func (j *JWTBuilder) WithCookieName(name string) *JWTBuilder {
	j.cookieName = name
	return j
}

func (j *JWTBuilder) WithCookiePath(path string) *JWTBuilder {
	j.cookiePath = path
	return j
}

func (j *JWTBuilder) WithCookieDomain(domain string) *JWTBuilder {
	j.cookieDomain = domain
	return j
}

func (j *JWTBuilder) Build() *jwt.Token {
	return jwt.NewWithClaims(j.signingMethod, j.mapClaims)
}

func (j *JWTBuilder) BuildSignedString() (string, error) {
	return j.Build().SignedString(j.secretKey)
}

func (j *JWTBuilder) BuildSignedStringT(t *testing.T) string {
	t.Helper()
	jwt, err := j.BuildSignedString()
	require.NoError(t, err, "failed to build signed JWT string")
	return jwt
}

func (j *JWTBuilder) BuildHTTPCookie() *http.Cookie {
	token, err := j.BuildSignedString()
	if err != nil {
		return nil
	}
	return &http.Cookie{
		Name:     j.cookieName,
		Value:    token,
		Path:     j.cookiePath,
		Domain:   j.cookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Until(j.tokenDuration.Time).Seconds()),
	}
}
