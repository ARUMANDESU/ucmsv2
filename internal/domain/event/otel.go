package event

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

type Otel struct {
	carrier map[string]string
}

func (o *Otel) Propagate(ctx context.Context) {
	if o.carrier == nil {
		o.carrier = make(map[string]string)
	}

	tcPropagator := propagation.TraceContext{}
	bgPropagator := propagation.Baggage{}

	tcPropagator.Inject(ctx, propagation.MapCarrier(o.carrier))
	bgPropagator.Inject(ctx, propagation.MapCarrier(o.carrier))
}

func (o *Otel) Extract() context.Context {
	if o.carrier == nil {
		o.carrier = make(map[string]string)
	}

	tcPropagator := propagation.TraceContext{}
	bgPropagator := propagation.Baggage{}

	ctx := context.Background()
	ctx = tcPropagator.Extract(ctx, propagation.MapCarrier(o.carrier))
	ctx = bgPropagator.Extract(ctx, propagation.MapCarrier(o.carrier))

	return ctx
}
