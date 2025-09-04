package staffhttp

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/go-chi/chi"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	staffapp "github.com/ARUMANDESU/ucms/internal/application/staff"
	"github.com/ARUMANDESU/ucms/internal/application/staff/cmd"
	"github.com/ARUMANDESU/ucms/internal/domain/staffinvitation"
	"github.com/ARUMANDESU/ucms/internal/domain/user"
	"github.com/ARUMANDESU/ucms/internal/ports/http/middlewares"
	"github.com/ARUMANDESU/ucms/pkg/ctxs"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	"github.com/ARUMANDESU/ucms/pkg/otelx"
	"github.com/ARUMANDESU/ucms/pkg/sanitizex"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
)

const (
	ISS               = "ucmsv2_invitation"
	InvitationSubject = "invitation_validation"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/staff")
	logger = otelslog.NewLogger("ucms/internal/ports/http/staff")
)

var (
	recipientsEmailRules = []validation.Rule{validation.Count(0, 100), validation.Each(validation.Required, is.Email)}
	validityRules        = []validation.Rule{validation.NilOrNotEmpty}
)

type HTTP struct {
	tracer                  trace.Tracer
	logger                  *slog.Logger
	cmd                     *staffapp.Command
	query                   *staffapp.Query
	errhandler              *httpx.ErrorHandler
	middleware              *middlewares.Middleware
	acceptInvitationPageURL string
	signingMethod           jwt.SigningMethod
	secretKey               string
	invitationTokenExp      time.Duration
}

type Args struct {
	Tracer                  trace.Tracer
	Logger                  *slog.Logger
	App                     *staffapp.App
	Errhandler              *httpx.ErrorHandler
	Middleware              *middlewares.Middleware
	AcceptInvitationPageURL string
	InvitationTokenAlg      jwt.SigningMethod
	InvitationTokenKey      string
	InvitationTokenExp      time.Duration
}

func NewHTTP(args Args) *HTTP {
	if args.App == nil {
		panic("app is required")
	}
	if args.Middleware == nil {
		panic("middleware is required")
	}
	if args.AcceptInvitationPageURL == "" {
		panic("accept invitation page url is required")
	}
	h := &HTTP{
		tracer:                  args.Tracer,
		logger:                  args.Logger,
		cmd:                     &args.App.Command,
		query:                   &args.App.Query,
		errhandler:              args.Errhandler,
		middleware:              args.Middleware,
		acceptInvitationPageURL: args.AcceptInvitationPageURL,
		signingMethod:           args.InvitationTokenAlg,
		secretKey:               args.InvitationTokenKey,
		invitationTokenExp:      args.InvitationTokenExp,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}
	if h.errhandler == nil {
		h.errhandler = httpx.NewErrorHandler()
	}
	if h.invitationTokenExp == 0 {
		h.invitationTokenExp = 15 * time.Minute
	}
	if h.signingMethod == nil {
		h.signingMethod = jwt.SigningMethodHS256
	}
	if h.secretKey == "" {
		panic("secret key is required for invitation token")
	}

	return h
}

func (h *HTTP) Route(r chi.Router) {
	r.Route("/v1/staffs", func(r chi.Router) {
		r.Use(h.middleware.Auth, h.middleware.StaffOnly)

		r.Route("/invitations", func(r chi.Router) {
			r.Post("/", h.CreateInvitation)
			r.Put("/{invitation_id}/recipients", h.UpdateInvitationRecipients)
			r.Put("/{invitation_id}/validity", h.UpdateInvitationValidity)
			r.Delete("/{invitation_id}", h.DeleteInvitation)
		})
	})

	r.Route("/v1/invitations", func(r chi.Router) {
		r.Get("/{invitation_code}/validate", h.Validate)
		r.Post("/accept", h.AcceptInvitation)
	})
}

type CreateInvitationRequest struct {
	Recipients []string   `json:"recipients_email"`
	ValidFrom  *time.Time `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until"`
}

func (c *CreateInvitationRequest) Sanitize() {
	c.Recipients = sanitizex.DeduplicateSlice(c.Recipients, sanitizex.StringTransformFunc(sanitizex.CleanSingleLine))
}

func (c *CreateInvitationRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{
		"request.recipients_count": len(c.Recipients),
		"request.valid_from":       c.ValidFrom,
		"request.valid_until":      c.ValidUntil,
	})
}

func (c *CreateInvitationRequest) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Recipients, recipientsEmailRules...),
		validation.Field(&c.ValidFrom, validityRules...),
		validation.Field(&c.ValidUntil, validityRules...),
	)
}

func (h *HTTP) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.CreateInvitation")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	var req CreateInvitationRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read body")
		return
	}

	req.Sanitize()
	req.SetSpanAttrs(span)
	err = req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "validation failed")
		return
	}

	err = h.cmd.CreateInvitation.Handle(ctx, cmd.CreateInvitation{
		CreatorID:       ctxUser.ID,
		RecipientsEmail: req.Recipients,
		ValidFrom:       req.ValidFrom,
		ValidUntil:      req.ValidUntil,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to create invitation")
		return
	}

	httpx.Success(w, r, http.StatusCreated, nil)
}

type UpdateInvitationRecipientsRequest struct {
	Recipients []string `json:"recipients_email"`
}

func (r *UpdateInvitationRecipientsRequest) Sanitize() {
	r.Recipients = sanitizex.DeduplicateSlice(r.Recipients, sanitizex.StringTransformFunc(sanitizex.CleanSingleLine))
}

func (r *UpdateInvitationRecipientsRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{"request.recipients_count": len(r.Recipients)})
}

func (r *UpdateInvitationRecipientsRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Recipients, recipientsEmailRules...),
	)
}

func (h *HTTP) UpdateInvitationRecipients(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.UpdateInvitationRecipients")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	invitationID, err := httpx.ReadUUIDUrlParam(r, "invitation_id")
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid invitation_id")
		return
	}
	span.SetAttributes(attribute.String("request.invitation_id", invitationID.String()))

	var req UpdateInvitationRecipientsRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read body")
		return
	}

	req.Sanitize()
	req.SetSpanAttrs(span)
	err = req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "validation failed")
		return
	}

	err = h.cmd.UpdateInvitationRecipients.Handle(ctx, cmd.UpdateInvitationRecipients{
		InvitationID:    staffinvitation.ID(invitationID),
		CreatorID:       ctxUser.ID,
		RecipientsEmail: req.Recipients,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to update invitation recipients")
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

type UpdateInvitationValidityRequest struct {
	ValidFrom  *time.Time `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until"`
}

func (r *UpdateInvitationValidityRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{
		"request.valid_from":  r.ValidFrom,
		"request.valid_until": r.ValidUntil,
	})
}

func (r *UpdateInvitationValidityRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.ValidFrom, validityRules...),
		validation.Field(&r.ValidUntil, validityRules...),
	)
}

func (h *HTTP) UpdateInvitationValidity(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.UpdateInvitationValidity")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	invitationID, err := httpx.ReadUUIDUrlParam(r, "invitation_id")
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid invitation_id")
		return
	}
	span.SetAttributes(attribute.String("request.invitation_id", invitationID.String()))

	var req UpdateInvitationValidityRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read body")
		return
	}

	req.SetSpanAttrs(span)
	err = req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "validation failed")
		return
	}

	err = h.cmd.UpdateInvitationValidity.Handle(ctx, cmd.UpdateInvitationValidity{
		InvitationID: staffinvitation.ID(invitationID),
		CreatorID:    ctxUser.ID,
		ValidFrom:    req.ValidFrom,
		ValidUntil:   req.ValidUntil,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to update invitation validity")
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.DeleteInvitation")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	invitationID, err := httpx.ReadUUIDUrlParam(r, "invitation_id")
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid invitation_id")
		return
	}
	span.SetAttributes(attribute.String("request.invitation_id", invitationID.String()))

	err = h.cmd.DeleteInvitation.Handle(ctx, cmd.DeleteInvitation{
		InvitationID: staffinvitation.ID(invitationID),
		CreatorID:    ctxUser.ID,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to delete invitation")
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) Validate(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.Validate")
	defer span.End()

	invitationCode := chi.URLParam(r, "invitation_code")
	invitationCode = sanitizex.CleanSingleLine(invitationCode)
	err := validation.Validate(invitationCode, validation.Required, validation.Length(1, 1000))
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid invitation_code")
		return
	}

	email := r.URL.Query().Get("email")
	email = sanitizex.CleanSingleLine(email)
	err = validation.Validate(email, validation.Required, is.EmailFormat)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid email")
		return
	}

	err = h.cmd.ValidateInvitation.Handle(ctx, cmd.ValidateInvitation{
		InvitationCode: invitationCode,
		Email:          email,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to validate invitation")
		return
	}

	signedToken, err := SignInvitationJWTToken(
		invitationCode,
		email,
		h.signingMethod,
		h.secretKey,
		h.invitationTokenExp,
	)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to sign invitation token")
		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s?token=%s", h.acceptInvitationPageURL, url.QueryEscape(signedToken)), http.StatusFound)
}

func SignInvitationJWTToken(
	invitationCode string,
	email string,
	signingMethod jwt.SigningMethod,
	secretKey string,
	expiration time.Duration,
) (string, error) {
	const op = "http.SignInvitationJWTToken"
	jwtToken := jwt.NewWithClaims(signingMethod, jwt.MapClaims{
		"iss":             ISS,
		"sub":             InvitationSubject,
		"exp":             time.Now().Add(expiration).Unix(),
		"invitation_code": invitationCode,
		"email":           email,
	})

	signedToken, err := jwtToken.SignedString([]byte(secretKey))
	if err != nil {
		return "", errorx.NewInternalError().WithCause(err, op)
	}
	return signedToken, nil
}

type AcceptInvitationRequest struct {
	Token     string `json:"token"`
	Barcode   string `json:"barcode"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (r *AcceptInvitationRequest) Sanitize() {
	r.Token = sanitizex.CleanSingleLine(r.Token)
	r.Barcode = sanitizex.CleanSingleLine(r.Barcode)
	r.Username = sanitizex.CleanSingleLine(r.Username)
	r.Password = strings.TrimSpace(r.Password)
	r.FirstName = sanitizex.CleanSingleLine(r.FirstName)
	r.LastName = sanitizex.CleanSingleLine(r.LastName)
}

func (r *AcceptInvitationRequest) SetSpanAttrs(span trace.Span) {
	otelx.SetSpanAttrs(span, map[string]any{
		"request.token":    r.Token,
		"request.username": logging.RedactUsername(r.Username),
	})
}

func (r *AcceptInvitationRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Token, validation.Required, validation.Length(1, 1000)),
		validation.Field(&r.Barcode, validation.Required, validation.Length(1, 80), is.Alphanumeric),
		validation.Field(&r.Username, validation.Required, validation.Length(2, 100), validationx.IsUsername),
		validation.Field(&r.Password, validationx.PasswordRules...),
		validation.Field(&r.FirstName, validationx.NameRules...),
		validation.Field(&r.LastName, validationx.NameRules...),
	)
}

func (h *HTTP) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.AcceptInvitation")
	defer span.End()

	var req AcceptInvitationRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read body")
		return
	}

	req.Sanitize()
	req.SetSpanAttrs(span)
	err := req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "validation failed")
		return
	}

	invitationCode, email, err := ParseInvitationJWTToken(req.Token, h.signingMethod, h.secretKey)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid or expired token")
		return
	}

	cmd := cmd.AcceptInvitation{
		InvitationCode: invitationCode,
		Email:          email,
		Barcode:        user.Barcode(req.Barcode),
		Username:       req.Username,
		Password:       req.Password,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
	}
	err = h.cmd.AcceptInvitation.Handle(ctx, cmd)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to accept invitation")
		return
	}

	httpx.Success(w, r, http.StatusCreated, nil)
}

func ParseInvitationJWTToken(tokenString string, signingMethod jwt.SigningMethod, secretKey string) (invitationCode string, email string, err error) {
	const op = "http.ParseInvitationJWTToken"
	jwtToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != signingMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	}, jwt.WithValidMethods([]string{signingMethod.Alg()}))
	if err != nil {
		return "", "", errorx.NewInvalidCredentials().WithCause(err, op)
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok || !jwtToken.Valid {
		return "", "", errorx.NewInvalidCredentials().WithCause(fmt.Errorf("invalid invitation token"), op)
	}
	if claims["iss"] != ISS || claims["sub"] != InvitationSubject {
		return "", "", errorx.NewInvalidCredentials().
			WithCause(fmt.Errorf("invalid invitation token issuer or subject: iss=%v, sub=%v", claims["iss"], claims["sub"]), op)
	}
	invitationCode, ok = claims["invitation_code"].(string)
	if !ok || invitationCode == "" {
		return "", "", errorx.NewInvalidCredentials().
			WithCause(fmt.Errorf("invitation_code not found or type assertion failed in invitation token claims: %T", claims["invitation_code"]), op)
	}
	email, ok = claims["email"].(string)
	if !ok || email == "" {
		return "", "", errorx.NewInvalidCredentials().
			WithCause(fmt.Errorf("email not found or type assertion failed in invitation token claims: %T", claims["email"]), op)
	}

	return invitationCode, email, nil
}
