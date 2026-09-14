package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/GagarinRu/avatars/internal/storage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Idempotency struct {
	store storage.Storage
}

func NewIdempotency(store storage.Storage) *Idempotency {
	return &Idempotency{store: store}
}

func (i *Idempotency) IsProcessed(ctx context.Context, messageID string) (bool, error) {
	return i.store.IsMessageProcessed(ctx, messageID)
}

func (i *Idempotency) MarkProcessed(ctx context.Context, messageID string) error {
	return i.store.MarkMessageProcessed(ctx, messageID)
}

type DeliveryHandler func(ctx context.Context, msg AvatarProcessMessage, d amqp.Delivery) error

type Consumer struct {
	ch   Channel
	idem *Idempotency
}

func NewConsumer(ch Channel, idem *Idempotency) *Consumer {
	return &Consumer{ch: ch, idem: idem}
}

func (c *Consumer) Consume(ctx context.Context, handler DeliveryHandler) error {
	if err := c.ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}
	deliveries, err := c.ch.Consume(QueueProcessing, "avatars-worker", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			c.handleDelivery(ctx, handler, d)
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, handler DeliveryHandler, d amqp.Delivery) {
	msgCtx := ExtractTraceContext(ctx, d.Headers)
	msgCtx, span := otel.Tracer("avatars-queue").Start(msgCtx, "consume_avatar_process")
	defer span.End()

	var msg AvatarProcessMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		_ = d.Nack(false, false)
		return
	}
	if msg.MessageID == "" {
		msg.MessageID = d.MessageId
	}
	span.SetAttributes(
		attribute.String("user_id", msg.UserID),
		attribute.String("message_id", msg.MessageID),
	)
	processed, err := c.idem.IsProcessed(msgCtx, msg.MessageID)
	if err != nil {
		_ = d.Nack(false, true)
		return
	}
	if processed {
		_ = d.Ack(false)
		return
	}
	if err := handler(msgCtx, msg, d); err != nil {
		retry := RetryCount(d.Headers) + 1
		if pubErr := SendToRetry(c.ch, d.Body, retry); pubErr != nil {
			_ = d.Nack(false, true)
			return
		}
		_ = d.Ack(false)
		return
	}
	if err := c.idem.MarkProcessed(msgCtx, msg.MessageID); err != nil {
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}
