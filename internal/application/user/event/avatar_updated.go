package userevent

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/avatars"
)

var (
	tracer = otel.Tracer("ucms/internal/application/user/event")
	logger = otelslog.NewLogger("ucms/internal/application/user/event")
)

type AvatarStorage interface {
	DeleteFile(ctx context.Context, key string) error
}

type AvatarUpdatedHandler struct {
	avatarStorage AvatarStorage
}

func NewAvatarUpdatedHandler(avatarStorage AvatarStorage) *AvatarUpdatedHandler {
	return &AvatarUpdatedHandler{
		avatarStorage: avatarStorage,
	}
}

func (h *AvatarUpdatedHandler) Handle(ctx context.Context, e *user.UserAvatarUpdated) error {
	ctx, span := tracer.Start(ctx, "AvatarUpdatedHandler.Handle",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(e.Extract())),
		trace.WithAttributes(
			attribute.String("event.user.id", e.UserID.String()),
			attribute.String("event.old_avatar.source", e.OldAvatar.Source.String()),
			attribute.String("event.old_avatar.s3_key", e.OldAvatar.S3Key),
			attribute.String("event.new_avatar.source", e.NewAvatar.Source.String()),
			attribute.String("event.new_avatar.s3_key", e.NewAvatar.S3Key),
		),
	)
	defer span.End()

	if e.OldAvatar.Source == avatars.SourceS3 && e.OldAvatar.S3Key != "" && e.OldAvatar.S3Key != e.NewAvatar.S3Key {
		err := h.avatarStorage.DeleteFile(ctx, e.OldAvatar.S3Key)
		if err != nil {
			logger.WarnContext(ctx, "failed to delete previous avatar from S3",
				slog.String("user_id", e.UserID.String()),
				slog.String("previous_s3_key", e.OldAvatar.S3Key),
				slog.String("error", err.Error()))
		} else {
			logger.DebugContext(ctx, "successfully deleted previous avatar from S3",
				slog.String("user_id", e.UserID.String()),
				slog.String("previous_s3_key", e.OldAvatar.S3Key))
		}
	}

	return nil
}
