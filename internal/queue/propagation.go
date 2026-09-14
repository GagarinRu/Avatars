package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

type amqpHeaderCarrier struct {
	headers amqp.Table
}

func (c *amqpHeaderCarrier) Get(key string) string {
	if c.headers == nil {
		return ""
	}
	raw, ok := c.headers[key]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func (c *amqpHeaderCarrier) Set(key, value string) {
	if c.headers == nil {
		c.headers = amqp.Table{}
	}
	c.headers[key] = value
}

func (c *amqpHeaderCarrier) Keys() []string {
	if c.headers == nil {
		return nil
	}
	keys := make([]string, 0, len(c.headers))
	for key := range c.headers {
		keys = append(keys, key)
	}
	return keys
}

// InjectTraceContext stores the active trace context in AMQP headers.
func InjectTraceContext(ctx context.Context, headers amqp.Table) amqp.Table {
	carrier := &amqpHeaderCarrier{headers: headers}
	if carrier.headers == nil {
		carrier.headers = amqp.Table{}
	}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.headers
}

// ExtractTraceContext restores trace context from AMQP headers.
func ExtractTraceContext(ctx context.Context, headers amqp.Table) context.Context {
	carrier := &amqpHeaderCarrier{headers: headers}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
