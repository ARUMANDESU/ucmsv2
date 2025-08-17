package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/role"
	authhttp "github.com/ARUMANDESU/ucms/internal/ports/http/auth"
	"github.com/ARUMANDESU/ucms/pkg/ctxs"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
)

var tracer = otel.Tracer("ucms/internal/ports/http/middleware")

func AuthMiddleware(secret []byte, exp time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), "AuthMiddleware")
			defer span.End()

			accessCookie, err := r.Cookie(authhttp.AccessJWTCookie)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to get access token from cookie")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}

			err = validation.Validate(accessCookie.Value, validation.Required, validation.Length(1, 1000))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to validate access token")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}

			accessToken, err := jwt.Parse(accessCookie.Value, func(t *jwt.Token) (any, error) {
				return secret, nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to parse access jwt token")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}
			if !accessToken.Valid {
				err = errorx.NewInvalidCredentials().WithCause(errors.New("invalid access token"))
				span.RecordError(err)
				span.SetStatus(codes.Error, "invalid access token")
				httpx.NewErrorHandler().HandleError(w, r, err)
				return
			}

			accessClaims, ok := accessToken.Claims.(jwt.MapClaims)
			if !ok {
				err = errorx.NewInvalidCredentials().WithCause(errors.New("failed to parse access token claims"))
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to parse access token claims")
				httpx.NewErrorHandler().HandleError(w, r, err)
				return
			}
			if accessClaims["iss"] != "ucmsv2_auth" || accessClaims["sub"] != "user" {
				err = errors.New("invalid access token issuer or subject")
				span.RecordError(err)
				span.SetStatus(codes.Error, "invalid access token claims")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}
			userRole, ok := accessClaims["user_role"].(string)
			if !ok {
				err = fmt.Errorf("role not found or type assertion failed in access token claims: %T", accessClaims["role"])
				span.RecordError(err)
				span.SetStatus(codes.Error, "role not found or type assertion failed in access token claims")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}
			if userRole == "" {
				err = errors.New("role is empty in access token claims")
				span.RecordError(err)
				span.SetStatus(codes.Error, "role is empty in access token claims")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}
			uid, ok := accessClaims["uid"].(string)
			if !ok {
				err = fmt.Errorf("user ID not found or type assertion failed in access token claims: %T", accessClaims["uid"])
				span.RecordError(err)
				span.SetStatus(codes.Error, "user ID not found or type assertion failed in access token claims")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}
			expUnix, ok := accessClaims["exp"].(float64)
			if !ok {
				span.RecordError(err)
				span.SetStatus(codes.Error, "invalid access token expiration")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
				return
			}
			exp := time.Unix(int64(expUnix), 0)
			if exp.Before(time.Now().UTC()) {
				span.RecordError(err)
				span.SetStatus(codes.Error, "access token expired")
				httpx.NewErrorHandler().HandleError(w, r, errorx.NewInvalidCredentials().WithCause(errors.New("access token expired")))
				return
			}

			ctx = ctxs.WithUser(ctx, &ctxs.User{
				ID:   uid,
				Role: role.Global(userRole),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
