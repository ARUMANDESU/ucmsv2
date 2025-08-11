package event

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

type Otel struct {
	Carrier map[string]string `json:"carrier,omitempty"`
}

func (o *Otel) Propagate(ctx context.Context) {
	if o.Carrier == nil {
		o.Carrier = make(map[string]string)
	}

	tcPropagator := propagation.TraceContext{}
	bgPropagator := propagation.Baggage{}

	tcPropagator.Inject(ctx, propagation.MapCarrier(o.Carrier))
	bgPropagator.Inject(ctx, propagation.MapCarrier(o.Carrier))
}

func (o *Otel) Extract() context.Context {
	if o.Carrier == nil {
		o.Carrier = make(map[string]string)
	}

	tcPropagator := propagation.TraceContext{}
	bgPropagator := propagation.Baggage{}

	ctx := context.Background()
	ctx = tcPropagator.Extract(ctx, propagation.MapCarrier(o.Carrier))
	ctx = bgPropagator.Extract(ctx, propagation.MapCarrier(o.Carrier))

	return ctx
}
