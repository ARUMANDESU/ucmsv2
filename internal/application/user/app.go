package userapp

import (
	usercmd "gitlab.com/ucmsv2/ucms-backend/internal/application/user/cmd"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
)

type App struct {
	Command Command
	Query   Query
}

type Command struct {
	UpdateAvatar *usercmd.UpdateAvatarHandler
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
				Repo:                args.UserRepo,
			}),
		},
		Query: Query{},
	}
}
