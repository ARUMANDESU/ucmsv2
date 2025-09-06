package watermillx

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/event"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/registration"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/staffinvitation"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
)

func NewEventProcessor(router *message.Router, conn *pgxpool.Pool, logger watermill.LoggerAdapter) (*cqrs.EventProcessor, error) {
	return cqrs.NewEventProcessorWithConfig(router, cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			evt, ok := params.EventHandler.NewEvent().(event.Event)
			if !ok {
				return "", fmt.Errorf("event handler %T does not implement event.Event", params.EventHandler.NewEvent())
			}
			return MessageTopic(evt)
		},
		SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return watermillSQL.NewSubscriber(
				watermillSQL.BeginnerFromPgx(conn),
				watermillSQL.SubscriberConfig{
					ConsumerGroup:    params.EventHandler.HandlerName(),
					SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
					OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
					InitializeSchema: true,
				},
				logger,
			)
		},
		Marshaler:         cqrs.JSONMarshaler{},
		Logger:            logger,
		OnHandle:          nil,
		AckOnUnknownEvent: true,
	})
}

func NewEventGroupProcessor(router *message.Router, conn *pgxpool.Pool, logger watermill.LoggerAdapter) (*cqrs.EventGroupProcessor, error) {
	return cqrs.NewEventGroupProcessorWithConfig(router, cqrs.EventGroupProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventGroupProcessorGenerateSubscribeTopicParams) (string, error) {
			evt, ok := params.EventGroupHandlers[0].NewEvent().(event.Event) // all handlers' events' stream names have to be the same
			if !ok {
				return "", fmt.Errorf("event %T does not implement event.Event", params.EventGroupHandlers[0].NewEvent())
			}

			return MessageTopic(evt)
		},
		SubscriberConstructor: func(params cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return watermillSQL.NewSubscriber(
				watermillSQL.BeginnerFromPgx(conn),
				watermillSQL.SubscriberConfig{
					ConsumerGroup:    params.EventGroupName,
					SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
					OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
					InitializeSchema: true,
				},
				logger,
			)
		},
		OnHandle:          nil,
		AckOnUnknownEvent: true,
		Marshaler:         cqrs.JSONMarshaler{},
		Logger:            logger,
	})
}

func NewEventGroupProcessorForTests(router *message.Router, conn *pgxpool.Pool, logger watermill.LoggerAdapter) (*cqrs.EventGroupProcessor, error) {
	return cqrs.NewEventGroupProcessorWithConfig(router, cqrs.EventGroupProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventGroupProcessorGenerateSubscribeTopicParams) (string, error) {
			evt, ok := params.EventGroupHandlers[0].NewEvent().(event.Event) // all handlers' events' stream names have to be the same
			if !ok {
				return "", fmt.Errorf("event %T does not implement event.Event", params.EventGroupHandlers[0].NewEvent())
			}

			return MessageTopic(evt)
		},
		SubscriberConstructor: func(params cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return watermillSQL.NewSubscriber(
				watermillSQL.BeginnerFromPgx(conn),
				watermillSQL.SubscriberConfig{
					ConsumerGroup:    params.EventGroupName,
					SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
					OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
					InitializeSchema: false,
					PollInterval:     time.Millisecond * 10,
					ResendInterval:   0,
					RetryInterval:    0,
				},
				logger,
			)
		},
		OnHandle:          nil,
		AckOnUnknownEvent: true,
		Marshaler:         cqrs.JSONMarshaler{},
		Logger:            logger,
	})
}

func NewEventProcessorForTests(router *message.Router, conn *pgxpool.Pool, logger watermill.LoggerAdapter) (*cqrs.EventProcessor, error) {
	return cqrs.NewEventProcessorWithConfig(router, cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			evt, ok := params.EventHandler.NewEvent().(event.Event)
			if !ok {
				return "", fmt.Errorf("event handler %T does not implement event.Event", params.EventHandler.NewEvent())
			}
			return MessageTopic(evt)
		},
		SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return watermillSQL.NewSubscriber(
				watermillSQL.BeginnerFromPgx(conn),
				watermillSQL.SubscriberConfig{
					ConsumerGroup:    params.EventHandler.HandlerName(),
					SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
					OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
					InitializeSchema: false,
					PollInterval:     time.Millisecond * 10,
					ResendInterval:   0,
					RetryInterval:    0,
				},
				logger,
			)
		},
		Marshaler:         cqrs.JSONMarshaler{},
		Logger:            logger,
		OnHandle:          nil,
		AckOnUnknownEvent: true,
	})
}

func NewTxEventBus(tx pgx.Tx, logger watermill.LoggerAdapter) (*cqrs.EventBus, error) {
	publisher, err := watermillSQL.NewPublisher(
		watermillSQL.TxFromPgx(tx),
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	eventBus, err := cqrs.NewEventBusWithConfig(publisher, cqrs.EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			evt, ok := params.Event.(event.Event)
			if !ok {
				return "", fmt.Errorf("event %T does not implement event.Event", params.Event)
			}

			return MessageTopic(evt)
		},
		Marshaler: cqrs.JSONMarshaler{},
		Logger:    logger,
		OnPublish: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create event bus: %w", err)
	}

	return eventBus, nil
}

func Publish(ctx context.Context, tx pgx.Tx, logger watermill.LoggerAdapter, evts ...event.Event) error {
	if len(evts) == 0 {
		return nil
	}

	eventBus, err := NewTxEventBus(tx, logger)
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}

	for _, evt := range evts {
		if err := eventBus.Publish(ctx, evt); err != nil {
			return fmt.Errorf("failed to publish event %T: %w", evt, err)
		}
	}

	return nil
}

func MessageTopic(event event.Event) (string, error) {
	streamName := event.GetStreamName()
	if streamName == "" {
		return "", fmt.Errorf("stream name is empty, event: %T", event)
	}

	return streamName, nil
}

func InitializeEventSchema(ctx context.Context, conn *pgxpool.Pool, logger watermill.LoggerAdapter) error {
	subscriber, err := watermillSQL.NewSubscriber(
		watermillSQL.BeginnerFromPgx(conn),
		watermillSQL.SubscriberConfig{
			SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
			InitializeSchema: true,
		},
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	events := []string{
		registration.EventStreamName,
		user.StudentEventStreamName,
		user.StaffEventStreamName,
		user.UserEventStreamName,
		staffinvitation.EventStreamName,
	}

	for _, eventStream := range events {
		if err := subscriber.SubscribeInitialize(eventStream); err != nil {
			return fmt.Errorf("failed to initialize event schema for %s: %w", eventStream, err)
		}
	}

	return nil
}
