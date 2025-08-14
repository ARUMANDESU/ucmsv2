package watermill

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ARUMANDESU/ucms/internal/application/mail"
	"github.com/ARUMANDESU/ucms/internal/application/registration"
	studentapp "github.com/ARUMANDESU/ucms/internal/application/student"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
)

type Port struct {
	eventProcessor      *cqrs.EventProcessor
	eventGroupProcessor *cqrs.EventGroupProcessor
	cmdProcessor        *cqrs.CommandProcessor
}

type AppEventHandlers struct {
	Registration registration.Event
	Mail         mail.Event
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
	if handlers.Mail.RegistrationStarted == nil {
		return fmt.Errorf("mail registration started handler is nil")
	}
	if handlers.Student.StudentRegistrationCompleted == nil {
		return fmt.Errorf("student registration completed handler is nil")
	}

	err := p.eventGroupProcessor.AddHandlersGroup("email-event-group",
		cqrs.NewEventHandler("OnRegistrationStarted", handlers.Mail.RegistrationStarted.Handle),
	)
	if err != nil {
		return fmt.Errorf("failed to add registration started handler: %w", err)
	}

	err = p.eventProcessor.AddHandlers(cqrs.NewEventHandler("OnStudentRegistrationCompleted", handlers.Student.StudentRegistrationCompleted.Handle))
	if err != nil {
		return fmt.Errorf("failed to add student event handler: %w", err)
	}

	return nil
}
