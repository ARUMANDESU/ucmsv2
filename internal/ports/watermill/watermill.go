package watermill

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"

	mailevent "gitlab.com/ucmsv2/ucms-backend/internal/application/mail/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/registration"
	studentapp "gitlab.com/ucmsv2/ucms-backend/internal/application/student"
	"gitlab.com/ucmsv2/ucms-backend/pkg/watermillx"
)

type Port struct {
	eventProcessor      *cqrs.EventProcessor
	eventGroupProcessor *cqrs.EventGroupProcessor
	cmdProcessor        *cqrs.CommandProcessor
}

type AppEventHandlers struct {
	Registration registration.Event
	Mail         *mailevent.MailEventHandler
	Student      studentapp.Event
}

func NewPort(router *message.Router, conn *pgxpool.Pool, wmlogger watermill.LoggerAdapter) (*Port, error) {
	eventProcessor, err := watermillx.NewEventProcessor(router, conn, wmlogger)
	if err != nil {
		return nil, err
	}
	eventGroupProcessor, err := watermillx.NewEventGroupProcessor(router, conn, wmlogger)
	if err != nil {
		return nil, err
	}

	return &Port{
		eventProcessor:      eventProcessor,
		eventGroupProcessor: eventGroupProcessor,
		cmdProcessor:        &cqrs.CommandProcessor{},
	}, nil
}

func NewPortForTest(router *message.Router, conn *pgxpool.Pool, wmlogger watermill.LoggerAdapter) (*Port, error) {
	eventProcessor, err := watermillx.NewEventProcessorForTests(router, conn, wmlogger)
	if err != nil {
		return nil, err
	}
	eventGroupProcessor, err := watermillx.NewEventGroupProcessorForTests(router, conn, wmlogger)
	if err != nil {
		return nil, err
	}

	return &Port{
		eventProcessor:      eventProcessor,
		eventGroupProcessor: eventGroupProcessor,
		cmdProcessor:        &cqrs.CommandProcessor{},
	}, nil
}

func (p *Port) Run(ctx context.Context, handlers AppEventHandlers) error {
	err := p.eventProcessor.AddHandlers(
		cqrs.NewEventHandler("MailOnRegistrationStarted", handlers.Mail.HandleRegistrationStarted),
		cqrs.NewEventHandler("MailOnVerificationCodeResent", handlers.Mail.HandleVerificationCodeResent),
		cqrs.NewEventHandler("MailOnStudentRegistered", handlers.Mail.HandleStudentRegistered),
		cqrs.NewEventHandler("MailOnStaffInvitationCreated", handlers.Mail.HandleStaffInvitationCreated),
		cqrs.NewEventHandler("MailOnStaffInvitationRecipientsUpdated", handlers.Mail.HandleStaffInvitationRecipientsUpdated),
		cqrs.NewEventHandler("MailOnStaffInvitationAccepted", handlers.Mail.HandleStaffInvitationAccepted),

		cqrs.NewEventHandler("RegistrationOnStudentRegistered", handlers.Registration.Registration.StudentHandle),
	)
	if err != nil {
		return fmt.Errorf("failed to add event handlers: %w", err)
	}

	return nil
}
