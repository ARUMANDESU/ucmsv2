package usercmd

import (
	"context"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
	"gitlab.com/ucmsv2/ucms-backend/pkg/otelx"
)

const (
	MaxAvatarSize = 5 * 1024 * 1024 // 5 MB
)

var tracer = otel.Tracer("ucms/internal/application/user/cmd")

type AvatarStorage interface {
	UploadFile(ctx context.Context, key string, file io.Reader, contentType string) error
	DeleteFile(ctx context.Context, key string) error
}

type UserRepo interface {
	UpdateUser(ctx context.Context, id user.ID, updateFn func(context.Context, *user.User) error) error
}

type UpdateAvatar struct {
	UserID      user.ID
	File        io.Reader
	Size        int64
	ContentType string
	Filename    string
}

type UpdateAvatarHandler struct {
	tracer        trace.Tracer
	avatarService *user.AvatarService
	storage       AvatarStorage
	repo          UserRepo
}

type UpdateAvatarHandlerArgs struct {
	Tracer              trace.Tracer
	AvatarDomainService *user.AvatarService
	Storage             AvatarStorage
	UserRepo            UserRepo
}

func NewUpdateAvatarHandler(args UpdateAvatarHandlerArgs) *UpdateAvatarHandler {
	if args.Tracer == nil {
		args.Tracer = tracer
	}

	return &UpdateAvatarHandler{
		tracer:        args.Tracer,
		avatarService: args.AvatarDomainService,
		storage:       args.Storage,
		repo:          args.UserRepo,
	}
}

func (h *UpdateAvatarHandler) Handle(ctx context.Context, cmd *UpdateAvatar) error {
	const op = "usercmd.UpdateAvatarHandler.Handle"
	ctx, span := h.tracer.Start(ctx, "UpdateAvatarHandler.Handle", trace.WithAttributes(
		attribute.String("user.id", cmd.UserID.String()),
		attribute.String("file.content_type", cmd.ContentType),
		attribute.Int64("file.size", cmd.Size),
		attribute.String("file.filename", cmd.Filename),
	))
	defer span.End()

	if err := h.avatarService.ValidateAvatarFile(cmd.ContentType, cmd.Size); err != nil {
		otelx.RecordSpanError(span, err, "invalid avatar file")
		return errorx.Wrap(err, op)
	}

	newS3Key := h.avatarService.GenerateS3Key(cmd.UserID)
	span.AddEvent("generated new S3 key", trace.WithAttributes(attribute.String("s3.key", newS3Key)))

	if err := h.storage.UploadFile(ctx, newS3Key, cmd.File, cmd.ContentType); err != nil {
		otelx.RecordSpanError(span, err, "failed to upload avatar to storage")
		return errorx.Wrap(err, op)
	}
	span.AddEvent("uploaded new avatar to storage", trace.WithAttributes(attribute.String("s3.key", newS3Key)))

	err := h.repo.UpdateUser(ctx, cmd.UserID, func(ctx context.Context, u *user.User) error {
		if err := u.SetAvatarFromS3(newS3Key); err != nil {
			return errorx.Wrap(err, op)
		}
		return nil
	})
	if err != nil {
		otelx.RecordSpanError(span, err, "failed to update user avatar")
		return errorx.Wrap(err, op)
	}

	return nil
}
