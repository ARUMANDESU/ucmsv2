package watermillx

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type OTelFilteredSlogLogger struct {
	logger     *slog.Logger
	minLevel   slog.Level
	otelLogger log.Logger
}

func NewOTelFilteredSlogLogger(logger *slog.Logger, minLevel slog.Level) watermill.LoggerAdapter {
	return &OTelFilteredSlogLogger{
		logger:     logger,
		minLevel:   minLevel,
		otelLogger: global.GetLoggerProvider().Logger("watermill"),
	}
}

func (l *OTelFilteredSlogLogger) shouldLog(level slog.Level) bool {
	ctx := context.Background()

	var otelSeverity log.Severity
	switch {
	case level >= slog.LevelError:
		otelSeverity = log.SeverityError
	case level >= slog.LevelWarn:
		otelSeverity = log.SeverityWarn
	case level >= slog.LevelInfo:
		otelSeverity = log.SeverityInfo
	case level >= slog.LevelDebug:
		otelSeverity = log.SeverityDebug
	default:
		otelSeverity = log.SeverityTrace
	}

	return l.otelLogger.Enabled(ctx, log.EnabledParameters{Severity: otelSeverity})
}

func (l *OTelFilteredSlogLogger) Error(msg string, err error, fields watermill.LogFields) {
	if l.shouldLog(slog.LevelError) {
		l.logger.ErrorContext(context.Background(), msg, l.fieldsToAttrs(fields, slog.Any("error", err))...)
	}
}

func (l *OTelFilteredSlogLogger) Info(msg string, fields watermill.LogFields) {
	if l.shouldLog(slog.LevelInfo) {
		l.logger.InfoContext(context.Background(), msg, l.fieldsToAttrs(fields)...)
	}
}

func (l *OTelFilteredSlogLogger) Debug(msg string, fields watermill.LogFields) {
	if l.shouldLog(slog.LevelDebug) {
		l.logger.DebugContext(context.Background(), msg, l.fieldsToAttrs(fields)...)
	}
}

func (l *OTelFilteredSlogLogger) Trace(msg string, fields watermill.LogFields) {
	if l.minLevel < slog.LevelDebug {
		l.logger.DebugContext(context.Background(), msg, l.fieldsToAttrs(fields)...)
	}
}

func (l *OTelFilteredSlogLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &OTelFilteredSlogLogger{
		logger:     l.logger.With(l.fieldsToAttrs(fields)...),
		minLevel:   l.minLevel,
		otelLogger: l.otelLogger,
	}
}

func (l *OTelFilteredSlogLogger) fieldsToAttrs(fields watermill.LogFields, extra ...slog.Attr) []any {
	attrs := make([]any, 0, len(fields)+len(extra))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	for _, attr := range extra {
		attrs = append(attrs, attr)
	}
	return attrs
}
