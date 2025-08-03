package watermill

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ARUMANDESU/ucms/pkg/watermillx"
)

type Port struct {
	eventProcessor      *cqrs.EventProcessor
	eventGroupProcessor *cqrs.EventGroupProcessor
	cmdProcessor        *cqrs.CommandProcessor
	conn                *pgxpool.Conn
}

type AppEventHandlers struct {
}

func NewPort(router *message.Router, conn *pgxpool.Conn, wmlogger watermill.LoggerAdapter) (*Port, error) {
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
		conn:                conn,
	}, nil
}

func (p *Port) Run(ctx context.Context, handlers AppEventHandlers) error {
	err := p.eventGroupProcessor.AddHandlersGroup(
		"email-event-group",
	)
	if err != nil {
		return fmt.Errorf("failed to add event group handler: %w", err)
	}

	return nil
}

func (p *Port) Close(_ context.Context) error {
	p.conn.Release()
	return nil
}
