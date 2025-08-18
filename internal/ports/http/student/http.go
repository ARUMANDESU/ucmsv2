package studenthttp

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	studentapp "github.com/ARUMANDESU/ucms/internal/application/student"
	"github.com/ARUMANDESU/ucms/internal/application/student/studentquery"
	"github.com/ARUMANDESU/ucms/internal/ports/http/middleware"
	"github.com/ARUMANDESU/ucms/pkg/ctxs"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/student")
	logger = otelslog.NewLogger("ucms/internal/ports/http/student")
)

type HTTP struct {
	tracer                 trace.Tracer
	logger                 *slog.Logger
	app                    *studentapp.App
	errhandler             *httpx.ErrorHandler
	accessTokenExpDuration time.Duration
	accessTokenSecretKey   []byte
}

type Args struct {
	Tracer trace.Tracer
	Logger *slog.Logger
	App    *studentapp.App
}

func NewHTTP(args Args) *HTTP {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &HTTP{
		tracer:                 args.Tracer,
		logger:                 args.Logger,
		app:                    args.App,
		errhandler:             httpx.NewErrorHandler(),
		accessTokenExpDuration: 30 * time.Minute,
		accessTokenSecretKey:   []byte("secret1"),
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Route("/v1/students", func(r chi.Router) {
		r.With(middleware.AuthMiddleware(h.accessTokenSecretKey, h.accessTokenExpDuration)).Get("/me", h.GetStudent)
	})
}

func (h *HTTP) GetStudent(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "GetStudent")
	defer span.End()

	ctxUser, ok := ctxs.UserFromCtx(ctx)
	if !ok {
		err := errors.New("user not found in context")
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user from context")
		h.errhandler.HandleError(w, r, errorx.NewInvalidCredentials().WithCause(err))
		return
	}

	res, err := h.app.Query.GetStudent.Handle(ctx, studentquery.GetStudent{Barcode: ctxUser.Barcode})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get student")
		h.errhandler.HandleError(w, r, err)
		return
	}
	if res == nil {
		err := errors.New("returned student is nil")
		span.RecordError(err)
		span.SetStatus(codes.Error, "returned student is nil")
		h.errhandler.HandleError(w, r, errorx.NewInternalError().WithCause(err))
		return
	}

	httpRes := struct {
		Barcode   string `json:"barcode"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		Group     struct {
			ID    string `json:"id"`
			Major string `json:"major"`
			Name  string `json:"name"`
			Year  string `json:"year"`
		} `json:"group"`
		RegisteredAt string `json:"registered_at"`
	}{
		Barcode:   res.Barcode,
		AvatarURL: res.AvatarURL,
		Email:     res.Email,
		FirstName: res.FirstName,
		LastName:  res.LastName,
		Role:      res.Role,
		Group: struct {
			ID    string `json:"id"`
			Major string `json:"major"`
			Name  string `json:"name"`
			Year  string `json:"year"`
		}{
			ID:    res.Group.ID,
			Major: res.Group.Major,
			Name:  res.Group.Name,
			Year:  res.Group.Year,
		},
		RegisteredAt: res.RegisteredAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	httpx.Success(w, r, http.StatusOK, httpx.Envelope{"student": httpRes})
}
