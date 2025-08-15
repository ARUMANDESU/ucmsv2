package authapp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

const (
	AccessTokenExpDuration  = 30 * time.Minute
	RefreshTokenExpDuration = 14 * 24 * time.Hour
)

var (
	tracer = otel.Tracer("ucms/internal/application/auth")
	logger = otelslog.NewLogger("ucms/internal/application/auth")
)

var (
	ErrWrongEmailOrBarcodeOrPassword = errorx.NewInvalidRequest().WithKey("wrong_email_or_barcode_or_password")
	ErrWrongEmailOrBarcodeFormat     = errorx.NewInvalidRequest().WithKey("wrong_email_or_barcode_format")
)

type UserGetter interface {
	GetUserByID(ctx context.Context, id user.ID) (*user.User, error)
	GetUserByEmail(ctx context.Context, email string) (*user.User, error)
}

type App struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	usergetter UserGetter

	accessTokenExpDuration  time.Duration
	refreshTokenExpDuration time.Duration
	accessTokenSecretKey    string
	refreshTokenSecretKey   string
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
		accessTokenSecretKey:    args.AccessTokenSecretKey,
		refreshTokenSecretKey:   args.RefreshTokenSecretKey,
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
	Password       string
}

type LoginResponse struct {
	AccessToken  string
	RefreshToken string
}

// LoginHandle handles user login logic and return access jwt token
func (a *App) LoginHandle(ctx context.Context, cmd Login) (LoginResponse, error) {
	ctx, span := a.tracer.Start(ctx, "App.LoginHandle")
	defer span.End()

	var isEmail bool
	var isBarcode bool

	var u *user.User
	var err error
	if isEmail {
		u, err = a.usergetter.GetUserByEmail(ctx, cmd.EmailOrBarcode)
	} else if isBarcode {
		u, err = a.usergetter.GetUserByID(ctx, user.ID(cmd.EmailOrBarcode))
	} else {
		return LoginResponse{}, ErrWrongEmailOrBarcodeFormat
	}
	if err != nil {
		if errorx.IsNotFound(err) {
			return LoginResponse{}, ErrWrongEmailOrBarcodeOrPassword
		}
		return LoginResponse{}, fmt.Errorf("failed to get user by email or barcode: %w", err)
	}

	err = u.ComparePassword(cmd.Password)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to compare password: %w", err)
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":       "ucmsv2_auth",
		"sub":       "user",
		"exp":       time.Now().Add(a.accessTokenExpDuration).UTC(),
		"iat":       time.Now().UTC(),
		"uid":       u.ID().String(),
		"user_role": u.Role().String(),
	})
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":   "ucmsv2_auth",
		"sub":   "refresh",
		"exp":   time.Now().Add(a.refreshTokenExpDuration).UTC(),
		"iat":   time.Now().UTC(),
		"jti":   uuid.New().String(),
		"uid":   u.ID().String(),
		"scope": "refresh",
	})

	accessjwt, err := accessToken.SignedString(a.accessTokenSecretKey)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to sign access token: %w", err)
	}
	refreshjwt, err := refreshToken.SignedString(a.refreshTokenSecretKey)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return LoginResponse{
		AccessToken:  accessjwt,
		RefreshToken: refreshjwt,
	}, nil
}

type Refresh struct {
	AccessToken  string
	RefreshToken string
}

func (a *App) RefreshHandle(ctx context.Context, cmd Refresh) (LoginResponse, error) {
	ctx, span := a.tracer.Start(ctx, "App.RefreshHandle")
	defer span.End()

	accessToken, err := jwt.Parse(cmd.RefreshToken, func(t *jwt.Token) (any, error) {
		return a.accessTokenSecretKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to parse access token: %w", err)
	}
	refreshToken, err := jwt.Parse(cmd.RefreshToken, func(t *jwt.Token) (any, error) {
		return a.refreshTokenSecretKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	refreshClaims, ok := refreshToken.Claims.(jwt.MapClaims)
	if !ok {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("invalid_refresh_token_claims")
	}
	if refreshClaims["iss"] != "ucmsv2_auth" || refreshClaims["sub"] != "refresh" {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("invalid_refresh_token_claims")
	}
	exp, ok := refreshClaims["exp"].(time.Time)
	if !ok {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("invalid_refresh_token_exp")
	}
	if exp.Before(time.Now().UTC()) {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("refresh_token_expired")
	}

	accessClaims, ok := accessToken.Claims.(jwt.MapClaims)
	if !ok {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("invalid_access_token_claims")
	}

	if accessClaims["iss"] != "ucmsv2_auth" || accessClaims["sub"] != "user" {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("invalid_access_token_claims")
	}

	uid, ok := accessClaims["uid"].(string)
	if !ok {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("invalid_access_token_uid")
	}
	userRole, ok := accessClaims["user_role"].(string)
	if !ok {
		return LoginResponse{}, errorx.NewInvalidRequest().WithKey("invalid_access_token_user_role")
	}

	accessToken = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":       "ucmsv2_auth",
		"sub":       "user",
		"exp":       time.Now().Add(a.accessTokenExpDuration).UTC(),
		"iat":       time.Now().UTC(),
		"uid":       uid,
		"user_role": userRole,
	})

	accessjwt, err := accessToken.SignedString(a.accessTokenSecretKey)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	return LoginResponse{
		AccessToken:  accessjwt,
		RefreshToken: cmd.RefreshToken, // keep the same refresh token
	}, nil
}
