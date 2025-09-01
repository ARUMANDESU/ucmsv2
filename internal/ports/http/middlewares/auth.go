package middlewares

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	authhttp "github.com/ARUMANDESU/ucms/internal/ports/http/auth"
	"github.com/ARUMANDESU/ucms/pkg/ctxs"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/middleware")
	logger = otelslog.NewLogger("ucms/internal/ports/http/middleware")
)

type Middleware struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	secret     []byte
	exp        time.Duration
	errhandler *httpx.ErrorHandler
}

type Args struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	Secret     []byte
	Exp        time.Duration
	Errhandler *httpx.ErrorHandler
}

func NewMiddleware(args Args) *Middleware {
	m := &Middleware{
		tracer:     args.Tracer,
		logger:     args.Logger,
		secret:     args.Secret,
		exp:        args.Exp,
		errhandler: args.Errhandler,
	}

	if m.tracer == nil {
		m.tracer = tracer
	}
	if m.logger == nil {
		m.logger = logger
	}
	if len(m.secret) == 0 {
		panic("secret key is required for auth middleware")
	}
	if m.exp == 0 {
		m.exp = authapp.AccessTokenExpDuration
	}
	if m.errhandler == nil {
		m.errhandler = httpx.NewErrorHandler()
	}
	return m
}

func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "AuthMiddleware")
		defer span.End()

		accessCookie, err := r.Cookie(authhttp.AccessJWTCookie)
		if err != nil {
			m.errhandler.HandleError(w, r, span, errorx.NewInvalidCredentials().WithCause(err), "failed to get access token cookie")
			return
		}

		err = validation.Validate(accessCookie.Value, validation.Required, validation.Length(1, 1000))
		if err != nil {
			m.errhandler.HandleError(w, r, span, errorx.NewInvalidCredentials().WithCause(err), "invalid access token cookie")
			return
		}

		accessToken, err := jwt.Parse(accessCookie.Value, func(t *jwt.Token) (any, error) {
			return m.secret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil {
			m.errhandler.HandleError(w, r, span, errorx.NewInvalidCredentials().WithCause(err), "failed to parse access token")
			return
		}
		if !accessToken.Valid {
			err = errorx.NewInvalidCredentials().WithCause(errors.New("invalid access token"))
			m.errhandler.HandleError(w, r, span, err, "invalid access token")
			return
		}

		accessClaims, ok := accessToken.Claims.(jwt.MapClaims)
		if !ok {
			err = errorx.NewInvalidCredentials().WithCause(errors.New("failed to parse access token claims"))
			m.errhandler.HandleError(w, r, span, err, "failed to parse access token claims")
			return
		}
		if accessClaims["iss"] != "ucmsv2_auth" || accessClaims["sub"] != "user" {
			err = errorx.NewInvalidCredentials().
				WithCause(fmt.Errorf("invalid access token issuer or subject: iss=%v, sub=%v", accessClaims["iss"], accessClaims["sub"]))
			m.errhandler.HandleError(w, r, span, err, "invalid access token issuer or subject")
			return
		}
		userRole, ok := accessClaims["user_role"].(string)
		if !ok {
			err = errorx.NewInvalidCredentials().
				WithCause(fmt.Errorf("role not found or type assertion failed in access token claims: %T", accessClaims["user_role"]))
			m.errhandler.HandleError(w, r, span, err, "role not found or type assertion failed in access token claims")
			return
		}
		if userRole == "" {
			err = errorx.NewInvalidCredentials().WithCause(errors.New("role is empty in access token claims"))
			m.errhandler.HandleError(w, r, span, err, "role is empty in access token claims")
			return
		}
		uid, ok := accessClaims["uid"].(string)
		if !ok {
			err = errorx.NewInvalidCredentials().
				WithCause(fmt.Errorf("user id not found or type assertion failed in access token claims: %T", accessClaims["uid"]))
			m.errhandler.HandleError(w, r, span, err, "user id not found or type assertion failed in access token claims")
			return
		}
		expUnix, ok := accessClaims["exp"].(float64)
		if !ok {
			err = errorx.NewInvalidCredentials().
				WithCause(fmt.Errorf("expiration time not found or type assertion failed in access token claims: %T", accessClaims["exp"]))
			m.errhandler.HandleError(w, r, span, err, "expiration time not found or type assertion failed in access token claims")
			return
		}
		exp := time.Unix(int64(expUnix), 0)
		if exp.Before(time.Now().UTC()) {
			err = errorx.NewInvalidCredentials().WithCause(errors.New("access token is expired"))
			m.errhandler.HandleError(w, r, span, err, "access token is expired")
			return
		}
		userID, err := uuid.Parse(uid)
		if err != nil {
			err = errorx.NewInvalidCredentials().WithCause(err)
			m.errhandler.HandleError(w, r, span, err, "failed to parse user id in access token claims")
			return
		}

		ctx = ctxs.WithUser(ctx, &ctxs.User{
			ID:   user.ID(userID),
			Role: role.Global(userRole),
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
