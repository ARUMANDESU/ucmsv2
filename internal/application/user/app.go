package userapp

import (
	usercmd "gitlab.com/ucmsv2/ucms-backend/internal/application/user/cmd"
	userevent "gitlab.com/ucmsv2/ucms-backend/internal/application/user/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
)

type App struct {
	Command Command
	Query   Query
	Event   Event
}

type Command struct {
	UpdateAvatar *usercmd.UpdateAvatarHandler
	DeleteAvatar *usercmd.DeleteAvatarHandler
}

type Event struct {
	AvatarUpdated *userevent.AvatarUpdatedHandler
}

type Query struct{}

type Args struct {
	S3BaseURL     string
	AvatarStorage usercmd.AvatarStorage
	UserRepo      usercmd.UserRepo
}

func NewApp(args Args) *App {
	return &App{
		Command: Command{
			UpdateAvatar: usercmd.NewUpdateAvatarHandler(usercmd.UpdateAvatarHandlerArgs{
				AvatarDomainService: &user.AvatarService{},
				Storage:             args.AvatarStorage,
				UserRepo:            args.UserRepo,
			}),
			DeleteAvatar: usercmd.NewDeleteAvatarHandler(usercmd.DeleteAVatarHandlerArgs{
				UserRepo: args.UserRepo,
			}),
		},
		Event: Event{
			AvatarUpdated: userevent.NewAvatarUpdatedHandler(args.AvatarStorage),
		},
		Query: Query{},
	}
}
