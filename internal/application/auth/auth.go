package authapp

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/i18nx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
)

const (
	AccessTokenExpDuration  = 30 * time.Minute
	RefreshTokenExpDuration = 14 * 24 * time.Hour
	ISS                     = "ucmsv2_auth"
	UserSubject             = "user"
	RefreshSubject          = "refresh"
	RefreshScope            = "refresh"
)

var (
	tracer = otel.Tracer("ucms/internal/application/auth")
	logger = otelslog.NewLogger("ucms/internal/application/auth")
)

var ErrWrongEmailOrBarcodeOrPassword = errorx.NewUnauthorized().WithKey(i18nx.KeyWrongEmailBarcodePassword)

type UserGetter interface {
	GetUserByID(ctx context.Context, id user.ID) (*user.User, error)
	GetUserByBarcode(ctx context.Context, barcode user.Barcode) (*user.User, error)
	GetUserByEmail(ctx context.Context, email string) (*user.User, error)
}

type App struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	usergetter UserGetter

	accessTokenExpDuration  time.Duration
	refreshTokenExpDuration time.Duration
	accessTokenSecretKey    []byte
	refreshTokenSecretKey   []byte
	signingMethod           *jwt.SigningMethodHMAC
}

type Args struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	UserGetter UserGetter

	AccessTokenSecretKey    string
	RefreshTokenSecretKey   string
	AccessTokenlExpDuration *time.Duration
	RefreshTokenExpDuration *time.Duration
}

func NewApp(args Args) *App {
	app := &App{
		tracer:     tracer,
		logger:     logger,
		usergetter: args.UserGetter,

		accessTokenExpDuration:  AccessTokenExpDuration,
		refreshTokenExpDuration: RefreshTokenExpDuration,
		accessTokenSecretKey:    []byte(args.AccessTokenSecretKey),
		refreshTokenSecretKey:   []byte(args.RefreshTokenSecretKey),
		signingMethod:           jwt.SigningMethodHS256,
	}

	if args.AccessTokenlExpDuration != nil {
		app.accessTokenExpDuration = *args.AccessTokenlExpDuration
	}
	if args.RefreshTokenExpDuration != nil {
		app.refreshTokenExpDuration = *args.RefreshTokenExpDuration
	}
	if args.Tracer != nil {
		app.tracer = args.Tracer
	}
	if args.Logger != nil {
		app.logger = args.Logger
	}

	return app
}

type Login struct {
	EmailOrBarcode string
	IsEmail        bool
	Password       string
}

type LoginResponse struct {
	AccessToken     string
	RefreshToken    string
	AccessTokenExp  time.Duration
	RefreshTokenExp time.Duration
}

// LoginHandle handles user login logic and return access jwt token
func (a *App) LoginHandle(ctx context.Context, cmd Login) (LoginResponse, error) {
	ctx, span := a.tracer.Start(
		ctx,
		"App.LoginHandle",
		trace.WithAttributes(
			attribute.Bool("is_email", cmd.IsEmail),
			attribute.String("signing_method", a.signingMethod.Alg()),
			attribute.String("access_token_exp_duration", a.accessTokenExpDuration.String()),
			attribute.String("refresh_token_exp_duration", a.refreshTokenExpDuration.String()),
		),
	)
	defer span.End()

	var (
		u   *user.User
		err error
	)
	if cmd.IsEmail {
		span.SetAttributes(attribute.String("user.email", logging.RedactEmail(cmd.EmailOrBarcode)))
		u, err = a.usergetter.GetUserByEmail(ctx, cmd.EmailOrBarcode)
	} else {
		span.SetAttributes(attribute.String("user.Barcode", cmd.EmailOrBarcode))
		u, err = a.usergetter.GetUserByBarcode(ctx, user.Barcode(cmd.EmailOrBarcode))
	}
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get user by email or barcode")
		if errorx.IsNotFound(err) {
			return LoginResponse{}, ErrWrongEmailOrBarcodeOrPassword.WithCause(err)
		}
		return LoginResponse{}, err
	}

	err = u.ComparePassword(cmd.Password)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to compare user password")
		return LoginResponse{}, ErrWrongEmailOrBarcodeOrPassword.WithCause(err)
	}

	accessToken := jwt.NewWithClaims(a.signingMethod, jwt.MapClaims{
		"iss":       ISS,
		"sub":       UserSubject,
		"exp":       time.Now().Add(a.accessTokenExpDuration).Unix(),
		"iat":       time.Now().Unix(),
		"uid":       u.ID().String(),
		"user_role": u.Role().String(),
	})
	refreshToken := jwt.NewWithClaims(a.signingMethod, jwt.MapClaims{
		"iss":   ISS,
		"sub":   RefreshSubject,
		"exp":   time.Now().Add(a.refreshTokenExpDuration).Unix(),
		"iat":   time.Now().Unix(),
		"jti":   uuid.New().String(),
		"uid":   u.ID().String(),
		"scope": RefreshScope,
	})

	accessjwt, err := accessToken.SignedString(a.accessTokenSecretKey)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to sign access token")
		return LoginResponse{}, err
	}
	refreshjwt, err := refreshToken.SignedString(a.refreshTokenSecretKey)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to sign refresh token")
		return LoginResponse{}, err
	}

	return LoginResponse{
		AccessToken:     accessjwt,
		RefreshToken:    refreshjwt,
		AccessTokenExp:  a.accessTokenExpDuration,
		RefreshTokenExp: a.refreshTokenExpDuration,
	}, nil
}

type Refresh struct {
	RefreshToken string
}

func (a *App) RefreshHandle(ctx context.Context, cmd Refresh) (LoginResponse, error) {
	ctx, span := a.tracer.Start(
		ctx,
		"App.RefreshHandle",
		trace.WithAttributes(
			attribute.String("signing_method", a.signingMethod.Alg()),
			attribute.String("access_token_exp_duration", a.accessTokenExpDuration.String()),
			attribute.String("refresh_token_exp_duration", a.refreshTokenExpDuration.String()),
		),
	)
	defer span.End()

	refreshToken, err := jwt.Parse(cmd.RefreshToken, func(t *jwt.Token) (any, error) {
		return a.refreshTokenSecretKey, nil
	}, jwt.WithValidMethods([]string{a.signingMethod.Alg()}))
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to parse refresh token")
		return LoginResponse{}, errorx.NewInvalidCredentials().WithCause(err)
	}

	refreshClaims, ok := refreshToken.Claims.(jwt.MapClaims)
	if !ok {
		otelx.RecordSpanError(span, err, "invalid refresh token claims type")
		return LoginResponse{}, errorx.NewInvalidCredentials().WithCause(err)
	}
	if refreshClaims["iss"] != ISS || refreshClaims["sub"] != RefreshSubject {
		err = errors.New("invalid refresh token issuer or subject")
		otelx.RecordSpanError(span, err, "invalid refresh token claims")
		return LoginResponse{}, errorx.NewInvalidCredentials().WithCause(err)
	}
	expUnix, ok := refreshClaims["exp"].(float64)
	if !ok {
		otelx.RecordSpanError(span, err, "invalid refresh token exp claim type")
		return LoginResponse{}, errorx.NewInvalidCredentials().WithCause(err)
	}
	exp := time.Unix(int64(expUnix), 0)
	if exp.Before(time.Now().UTC()) {
		otelx.RecordSpanError(span, err, "refresh token is expired")
		return LoginResponse{}, errorx.NewInvalidCredentials().WithCause(err)
	}
	uid, ok := refreshClaims["uid"].(string)
	if !ok {
		err := errors.New("missing or invalid user id in refresh token claims")
		otelx.RecordSpanError(span, err, "invalid refresh token uid claim type")
		return LoginResponse{}, errorx.NewInvalidCredentials().WithCause(err)
	}
	span.SetAttributes(attribute.String("uid", uid))

	userID, err := uuid.Parse(uid)
	if err != nil {
		otelx.RecordSpanError(span, err, "invalid user id format in refresh token claims")
		return LoginResponse{}, errorx.NewInvalidCredentials().WithCause(err)
	}

	u, err := a.usergetter.GetUserByID(ctx, user.ID(userID))
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to get user by id from refresh token claims")
		return LoginResponse{}, errorx.NewInternalError().WithCause(err)
	}

	accessToken := jwt.NewWithClaims(a.signingMethod, jwt.MapClaims{
		"iss":       ISS,
		"sub":       UserSubject,
		"exp":       time.Now().Add(a.accessTokenExpDuration).Unix(),
		"iat":       time.Now().Unix(),
		"uid":       u.ID().String(),
		"user_role": u.Role().String(),
	})

	accessjwt, err := accessToken.SignedString(a.accessTokenSecretKey)
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to sign access token")
		return LoginResponse{}, errorx.NewInternalError().WithCause(err)
	}

	return LoginResponse{
		AccessToken:     accessjwt,
		RefreshToken:    cmd.RefreshToken, // keep the same refresh token
		AccessTokenExp:  a.accessTokenExpDuration,
		RefreshTokenExp: a.refreshTokenExpDuration,
	}, nil
}

type JWTTokenAssertion struct {
	token    string
	jwttoken *jwt.Token
	claims   jwt.MapClaims
	t        *testing.T
}

func NewJWTTokenAssertion(t *testing.T, token string, secretkey []byte) *JWTTokenAssertion {
	t.Helper()

	jwttoken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return secretkey, nil
	})
	require.NoError(t, err)

	claims, ok := jwttoken.Claims.(jwt.MapClaims)
	require.True(t, ok, "jwt token claims must be type jwt.MapClaims")

	return &JWTTokenAssertion{
		t:        t,
		token:    token,
		jwttoken: jwttoken,
		claims:   claims,
	}
}

func (a *JWTTokenAssertion) AssertValid() *JWTTokenAssertion {
	a.t.Helper()
	assert.NotNil(a.t, a.jwttoken, "jwt token should not be nil")
	assert.True(a.t, a.jwttoken.Valid, "jwt token should be valid")
	return a
}

func (a *JWTTokenAssertion) AssertNotValid() *JWTTokenAssertion {
	a.t.Helper()
	assert.NotNil(a.t, a.jwttoken, "jwt token should not be nil")
	assert.False(a.t, a.jwttoken.Valid, "jwt token should not be valid")
	return a
}

func (a *JWTTokenAssertion) AssertISS(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["iss"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertSub(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["sub"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertExp(expected time.Time) *JWTTokenAssertion {
	a.t.Helper()
	exp, ok := a.claims["exp"].(float64)
	require.True(a.t, ok, "exp claim must be of type float64, got %T", a.claims["exp"])
	assert.NotZero(a.t, exp, "exp claim should not be zero")
	expTime := time.Unix(int64(exp), 0)
	assert.WithinDuration(a.t, expected, expTime, time.Second, "exp claim should be within 1 second of expected time")
	return a
}

func (a *JWTTokenAssertion) AssertIAT(expected time.Time) *JWTTokenAssertion {
	a.t.Helper()
	iat, ok := a.claims["iat"].(float64)
	require.True(a.t, ok, "iat claim must be of type float64, got %T", a.claims["iat"])

	assert.NotZero(a.t, iat, "iat claim should not be zero")
	iatTime := time.Unix(int64(iat), 0)

	assert.WithinDuration(a.t, expected, iatTime, time.Second, "iat claim should be within 1 second of expected time")
	return a
}

func (a *JWTTokenAssertion) AssertScope(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["scope"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertJTI(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["jti"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertJTINotEmpty() *JWTTokenAssertion {
	a.t.Helper()
	assert.NotEmpty(a.t, a.claims["jti"], "jti claim should not be empty")
	return a
}

func (a *JWTTokenAssertion) AssertUID(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["uid"], expected)
	return a
}

func (a *JWTTokenAssertion) AssertUserRole(expected string) *JWTTokenAssertion {
	a.t.Helper()
	assert.Equal(a.t, a.claims["user_role"], expected)
	return a
}
