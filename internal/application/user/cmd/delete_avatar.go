package usercmd

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
)

type DeleteAvatar struct {
	UserID user.ID
}

type DeleteAvatarHandler struct {
	tracer trace.Tracer
	Repo   UserRepo
}

type DeleteAVatarHandlerArgs struct {
	Tracer   trace.Tracer
	UserRepo UserRepo
}

func NewDeleteAvatarHandler(args DeleteAVatarHandlerArgs) *DeleteAvatarHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}

	return &DeleteAvatarHandler{
		tracer: args.Tracer,
		Repo:   args.UserRepo,
	}
}

func (h *DeleteAvatarHandler) Handle(ctx context.Context, cmd *DeleteAvatar) error {
	const op = "usercmd.DeleteAvatarHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "DeleteAvatarHandler.Handle", trace.WithAttributes(
		attribute.String("user.id", cmd.UserID.String()),
	))
	defer span.End()

	err := h.Repo.UpdateUser(ctx, cmd.UserID, func(ctx context.Context, u *user.User) error {
		if err := u.DeleteAvatar(); err != nil {
			return errorx.Wrap(err, op)
		}
		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to delete user avatar")
		return errorx.Wrap(err, op)
	}

	return nil
}
