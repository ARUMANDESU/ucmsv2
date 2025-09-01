package staffapp

import "github.com/ARUMANDESU/ucms/internal/application/staff/cmd"

type App struct {
	Command Command
	Query   Query
}

type Command struct {
	CreateInvitation           *cmd.CreateInvitationHandler
	UpdateInvitationRecipients *cmd.UpdateInvitationRecipientsHandler
	UpdateInvitationValidity   *cmd.UpdateInvitationValidityHandler
	DeleteInvitation           *cmd.DeleteInvitationHandler
}

type Query struct{}

type Args struct {
	StaffInvitationRepo cmd.StaffInvitationRepo
}

func NewApp(args Args) *App {
	return &App{
		Command: Command{
			CreateInvitation: cmd.NewCreateInvitationHandler(
				cmd.CreateInvitationHandlerArgs{StaffInvitationRepo: args.StaffInvitationRepo},
			),
			UpdateInvitationRecipients: cmd.NewUpdateInvitationRecipientsHandler(
				cmd.UpdateInvitationRecipientsHandlerArgs{StaffInvitationRepo: args.StaffInvitationRepo},
			),
			UpdateInvitationValidity: cmd.NewUpdateInvitationValidityHandler(
				cmd.UpdateInvitationValidityHandlerArgs{StaffInvitationRepo: args.StaffInvitationRepo},
			),
			DeleteInvitation: cmd.NewDeleteInvitationHandler(
				cmd.DeleteInvitationHandlerArgs{StaffInvitationRepo: args.StaffInvitationRepo},
			),
		},
		Query: Query{},
	}
}
