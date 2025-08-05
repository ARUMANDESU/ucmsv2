package postgres

import (
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
)

var (
	tracer = otel.Tracer("ucms/internal/adapters/repos/postgres")
	logger = otelslog.NewLogger("ucms/internal/adapters/repos/postgres")
)
