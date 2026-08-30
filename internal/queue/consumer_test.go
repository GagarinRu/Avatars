package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/GagarinRu/avatars/internal/storage"
)

func TestHandleDeliverySkipsProcessed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()
	if err := store.MarkMessageProcessed(ctx, "done-id"); err != nil {
		t.Fatalf("mark: %v", err)
	}
	consumer := NewConsumer(nil, NewIdempotency(store))

	body, _ := json.Marshal(AvatarProcessMessage{MessageID: "done-id", UserID: "u1"})
	delivery := amqp.Delivery{Body: body, MessageId: "done-id"}

	called := false
	consumer.handleDelivery(ctx, func(context.Context, AvatarProcessMessage, amqp.Delivery) error {
		called = true
		return nil
	}, delivery)

	if called {
		t.Fatal("handler should not run for processed message")
	}
}

func TestHandleDeliverySuccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()
	consumer := NewConsumer(nil, NewIdempotency(store))

	body, err := json.Marshal(AvatarProcessMessage{MessageID: "new-id", UserID: "u1", StagingKey: "s"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	consumer.handleDelivery(ctx, func(context.Context, AvatarProcessMessage, amqp.Delivery) error {
		return nil
	}, amqp.Delivery{Body: body, MessageId: "new-id"})

	processed, err := store.IsMessageProcessed(ctx, "new-id")
	if err != nil || !processed {
		t.Fatalf("expected processed, got %v err=%v", processed, err)
	}
}

func TestHandleDeliveryInvalidJSON(t *testing.T) {
	t.Parallel()
	consumer := NewConsumer(nil, NewIdempotency(storage.NewMemStorage()))
	called := false
	consumer.handleDelivery(context.Background(), func(context.Context, AvatarProcessMessage, amqp.Delivery) error {
		called = true
		return nil
	}, amqp.Delivery{Body: []byte("{")})
	if called {
		t.Fatal("handler should not run for invalid json")
	}
}

func TestHandleDeliveryHandlerError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()
	ch := &recordingPublishChannel{}
	consumer := NewConsumer(ch, NewIdempotency(store))

	body, _ := json.Marshal(AvatarProcessMessage{MessageID: "err-id", UserID: "u1"})
	consumer.handleDelivery(ctx, func(context.Context, AvatarProcessMessage, amqp.Delivery) error {
		return errTest
	}, amqp.Delivery{Body: body, Headers: nil})

	if ch.lastQueue != QueueRetry30s {
		t.Fatalf("expected retry queue, got %q", ch.lastQueue)
	}
	processed, _ := store.IsMessageProcessed(ctx, "err-id")
	if processed {
		t.Fatal("should not mark processed on error")
	}
}

var errTest = errors.New("test error")

type recordingPublishChannel struct {
	lastQueue    string
	lastExchange string
	lastKey      string
}

func (r *recordingPublishChannel) Publish(exchange, key string, _, _ bool, msg amqp.Publishing) error {
	r.lastExchange = exchange
	r.lastKey = key
	if exchange == "" {
		r.lastQueue = key
	}
	return nil
}

func (r *recordingPublishChannel) Qos(int, int, bool) error { return nil }
func (r *recordingPublishChannel) Consume(string, string, bool, bool, bool, bool, amqp.Table) (<-chan amqp.Delivery, error) {
	return nil, nil
}
