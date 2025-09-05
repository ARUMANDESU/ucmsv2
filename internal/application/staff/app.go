package staffapp

import "gitlab.com/ucmsv2/ucms-backend/internal/application/staff/cmd"

type App struct {
	Command Command
	Query   Query
}

type Command struct {
	CreateInvitation           *cmd.CreateInvitationHandler
	UpdateInvitationRecipients *cmd.UpdateInvitationRecipientsHandler
	UpdateInvitationValidity   *cmd.UpdateInvitationValidityHandler
	DeleteInvitation           *cmd.DeleteInvitationHandler
	ValidateInvitation         *cmd.ValidateInvitationHandler
	AcceptInvitation           *cmd.AcceptInvitationHandler
}

type Query struct{}

type Args struct {
	StaffInvitationRepo cmd.StaffInvitationRepo
	StaffRepo           cmd.StaffRepo
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
			ValidateInvitation: cmd.NewValidateInvitationHandler(
				cmd.ValidateInvitationHandlerArgs{StaffInvitationRepo: args.StaffInvitationRepo},
			),
			AcceptInvitation: cmd.NewAcceptInvitationHandler(
				cmd.AcceptInvitationHandlerArgs{
					StaffInvitationRepo: args.StaffInvitationRepo,
					StaffRepo:           args.StaffRepo,
				},
			),
		},
		Query: Query{},
	}
}
