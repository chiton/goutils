package correlation

import (
	"context"

	"github.com/teris-io/shortid"
)

const (
	CorrelationIdKey       string = "correlation-id"
	ClientCorrelationIdKey string = "client-correlation-id"
)

// NewContext creates a new context enriched with the correlationId
func NewContext(ctx context.Context, correlationId string) context.Context {
	return context.WithValue(ctx, CorrelationIdKey, correlationId)
}

// FromContext returns a correlationId from the given context. If correlationId
// is not found in the context, this returns uuid.New().String()
func FromContext(ctx context.Context) string {
	if correlationId, ok := ctx.Value(CorrelationIdKey).(string); ok {
		return correlationId
	}

	correlationId, _ := shortid.Generate()
	return correlationId
}

// NewContextWithClientCorrelationId creates a new context enriched with the client correlationId.
// A client correlation id is provided by the client in the X-Correlation-Id request header.
func NewContextWithClientCorrelationId(ctx context.Context, clientCorrelationId string) context.Context {
	return context.WithValue(ctx, ClientCorrelationIdKey, clientCorrelationId)
}

// FromContextWithClientCorrelationId returns a client correlationId from the given context. If correlationId
// is not found in the context, this returns empty string
func FromContextWithClientCorrelationId(ctx context.Context) string {
	if correlationId, ok := ctx.Value(ClientCorrelationIdKey).(string); ok {
		return correlationId
	}

	return ""
}
