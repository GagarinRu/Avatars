package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func resolveRetryTarget(retryCount int) (queueName, exchange, routingKey string) {
	switch retryCount {
	case 1:
		return QueueRetry30s, "", ""
	case 2:
		return QueueRetry2m, "", ""
	default:
		return "", ExchangeDLX, RoutingDLQ
	}
}

func SendToRetry(ch publishChannel, body []byte, retryCount int) error {
	queueName, exchange, routingKey := resolveRetryTarget(retryCount)
	if queueName != "" {
		return ch.Publish("", queueName, false, false, amqp.Publishing{
			Body:    body,
			Headers: amqp.Table{"x-retry-count": retryCount},
		})
	}
	return ch.Publish(exchange, routingKey, false, false, amqp.Publishing{Body: body})
}

func RetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	raw, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

type publishChannel interface {
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type Publisher struct {
	ch publishChannel
}

func NewPublisher(ch publishChannel) *Publisher {
	return &Publisher{ch: ch}
}

func (p *Publisher) PublishAvatarProcess(ctx context.Context, msg AvatarProcessMessage) error {
	ctx, span := otel.Tracer("avatars-queue").Start(ctx, "publish_avatar_process")
	defer span.End()
	span.SetAttributes(
		attribute.String("user_id", msg.UserID),
		attribute.String("message_id", msg.MessageID),
	)

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	headers := InjectTraceContext(ctx, amqp.Table{})
	return p.ch.Publish(ExchangeDirect, RoutingUpload, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		MessageId:    msg.MessageID,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
	})
}
