package studenthttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	studentapp "github.com/ARUMANDESU/ucms/internal/application/student"
	"github.com/ARUMANDESU/ucms/internal/application/student/studentquery"
	"github.com/ARUMANDESU/ucms/internal/ports/http/middlewares"
	"github.com/ARUMANDESU/ucms/pkg/ctxs"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/student")
	logger = otelslog.NewLogger("ucms/internal/ports/http/student")
)

type HTTP struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	app        *studentapp.App
	middleware *middlewares.Middleware
	errhandler *httpx.ErrorHandler
}

type Args struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	App        *studentapp.App
	Middleware *middlewares.Middleware
	Errhandler *httpx.ErrorHandler
}

func NewHTTP(args Args) *HTTP {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &HTTP{
		tracer:     args.Tracer,
		logger:     args.Logger,
		app:        args.App,
		middleware: args.Middleware,
		errhandler: args.Errhandler,
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Route("/v1/students", func(r chi.Router) {
		r.With(h.middleware.Auth).Get("/me", h.GetStudent)
	})
}

type GetStudentResponse struct {
	Barcode      string    `json:"barcode"`
	AvatarURL    string    `json:"avatar_url"`
	Email        string    `json:"email"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Role         string    `json:"role"`
	Group        GroupInfo `json:"group"`
	RegisteredAt string    `json:"registered_at"`
}

type GroupInfo struct {
	ID    string `json:"id"`
	Major string `json:"major"`
	Name  string `json:"name"`
	Year  string `json:"year"`
}

func (h *HTTP) GetStudent(w http.ResponseWriter, r *http.Request) {
	const op = "studenthttp.HTTP.GetStudent"
	ctx, span := h.tracer.Start(r.Context(), "GetStudent")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	res, err := h.app.Query.GetStudent.Handle(ctx, studentquery.GetStudent{ID: ctxUser.ID})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get student")
		return
	}
	if res == nil {
		err := errors.New("returned student is nil")
		h.errhandler.HandleError(w, r, span, errorx.NewInternalError().WithCause(err, op), "failed to get student")
		return
	}

	httpRes := GetStudentResponse{
		Barcode:   res.Barcode,
		AvatarURL: res.AvatarURL,
		Email:     res.Email,
		FirstName: res.FirstName,
		LastName:  res.LastName,
		Role:      res.Role,
		Group: GroupInfo{
			ID:    res.Group.ID,
			Major: res.Group.Major,
			Name:  res.Group.Name,
			Year:  res.Group.Year,
		},
		RegisteredAt: res.RegisteredAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	httpx.Success(w, r, http.StatusOK, httpx.Envelope{"student": httpRes})
}
