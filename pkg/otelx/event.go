package otelx

import "context"

type OtelTracePropagator interface {
	Propagate(ctx context.Context)
}

type OtelTraceExtractor interface {
	Extract() context.Context
}

type OtelTracePropagatorExtractor interface {
	OtelTracePropagator
	OtelTraceExtractor
}

func ContextFromExtractor(extractor OtelTraceExtractor) context.Context {
	if extractor == nil {
		return context.Background()
	}
	return extractor.Extract()
}
