package userhttp

import (
	"log/slog"
	"net/http"

	"github.com/ARUMANDESU/validation"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	userapp "gitlab.com/ucmsv2/ucms-backend/internal/application/user"
	usercmd "gitlab.com/ucmsv2/ucms-backend/internal/application/user/cmd"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/ports/http/middlewares"
	"gitlab.com/ucmsv2/ucms-backend/pkg/ctxs"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/httpx"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/user")
	logger = otelslog.NewLogger("ucms/internal/ports/http/user")
)

type HTTP struct {
	tracer     trace.Tracer
	logger     *slog.Logger
	cmd        userapp.Command
	middleware *middlewares.Middleware
	errhandler *httpx.ErrorHandler
}

type Args struct {
	Tracer     trace.Tracer
	Logger     *slog.Logger
	UserApp    *userapp.App
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
		cmd:        args.UserApp.Command,
		middleware: args.Middleware,
		errhandler: args.Errhandler,
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Route("/v1/users", func(r chi.Router) {
		r.Use(h.middleware.Auth)

		r.Patch("/me/avatar", h.UpdateAvatar)
		r.Delete("/me/avatar", h.DeleteAvatar)
	})
}

func (h *HTTP) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	const op = "user.HTTP.UpdateAvatar"
	ctx, span := h.tracer.Start(r.Context(), op)
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}
	ctxUser.SetSpanAttrs(span)

	err = r.ParseMultipartForm(usercmd.MaxAvatarSize)
	if err != nil {
		err = errorx.NewInvalidRequest().WithCause(err, op)
		h.errhandler.HandleError(w, r, span, err, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		err = errorx.NewInvalidRequest().WithCause(err, op)
		h.errhandler.HandleError(w, r, span, err, "failed to get avatar file from form")
		return
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			h.logger.Warn("failed to close avatar file", slog.String("error", cerr.Error()))
		}
	}()

	err = validation.Validate(
		header.Size,
		validation.Max(usercmd.MaxAvatarSize).ErrorObject(user.ErrAvatarTooLarge),
	)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "invalid avatar file")
		return
	}

	cmd := &usercmd.UpdateAvatar{
		UserID:      ctxUser.ID,
		File:        file,
		Size:        header.Size,
		ContentType: header.Header.Get("Content-Type"),
		Filename:    header.Filename,
	}

	err = h.cmd.UpdateAvatar.Handle(ctx, cmd)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to update avatar")
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "HTTP.DeleteAvatar")
	defer span.End()

	ctxUser, err := ctxs.UserFromCtx(ctx)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get user from context")
		return
	}

	ctxUser.SetSpanAttrs(span)

	if err := h.cmd.DeleteAvatar.Handle(ctx, &usercmd.DeleteAvatar{UserID: ctxUser.ID}); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to delete user avatar")
		return
	}

	httpx.Success(w, r, http.StatusOK, nil)
}
