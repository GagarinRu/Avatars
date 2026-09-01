package queue_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRetryCount(t *testing.T) {
	t.Parallel()
	if got := queue.RetryCount(nil); got != 0 {
		t.Fatalf("got %d", got)
	}
	headers := amqp.Table{"x-retry-count": int32(2)}
	if got := queue.RetryCount(headers); got != 2 {
		t.Fatalf("got %d", got)
	}
}

func TestResolveRetryTarget(t *testing.T) {
	t.Parallel()
	cases := []struct {
		count           int
		queue, exchange string
	}{
		{1, queue.QueueRetry30s, ""},
		{2, queue.QueueRetry2m, ""},
		{3, "", queue.ExchangeDLX},
	}
	for _, tc := range cases {
		q, ex, rk := queue.ResolveRetryTargetForTest(tc.count)
		if q != tc.queue || ex != tc.exchange {
			t.Fatalf("count %d: queue=%q exchange=%q rk=%q", tc.count, q, ex, rk)
		}
		if tc.count >= 3 && rk != queue.RoutingDLQ {
			t.Fatalf("expected dlq routing key")
		}
	}
}

func TestAvatarProcessMessageJSON(t *testing.T) {
	t.Parallel()
	msg := queue.AvatarProcessMessage{
		MessageID:  "id-1",
		UserID:     "user-1",
		StagingKey: "staging/key",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded queue.AvatarProcessMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.MessageID != msg.MessageID || decoded.UserID != msg.UserID {
		t.Fatalf("unexpected: %+v", decoded)
	}
}

func TestIdempotencyStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()
	idem := queue.NewIdempotency(store)

	ok, err := idem.IsProcessed(ctx, "m1")
	if err != nil || ok {
		t.Fatalf("expected not processed")
	}
	if err := idem.MarkProcessed(ctx, "m1"); err != nil {
		t.Fatalf("mark: %v", err)
	}
	ok, err = idem.IsProcessed(ctx, "m1")
	if err != nil || !ok {
		t.Fatalf("expected processed")
	}
}

func TestPublisherPublishAvatarProcess(t *testing.T) {
	t.Parallel()
	ch := &recordingChannel{}
	pub := queue.NewPublisher(ch)
	err := pub.PublishAvatarProcess(context.Background(), queue.AvatarProcessMessage{
		MessageID:  "mid",
		UserID:     "u1",
		StagingKey: "staging/u1/mid",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if ch.lastExchange != queue.ExchangeDirect || ch.lastKey != queue.RoutingUpload {
		t.Fatalf("unexpected publish: %s %s", ch.lastExchange, ch.lastKey)
	}
	if ch.lastMsgID != "mid" {
		t.Fatalf("message id = %q", ch.lastMsgID)
	}
}

type recordingChannel struct {
	lastExchange string
	lastKey      string
	lastMsgID    string
}

func (r *recordingChannel) Publish(exchange, key string, _, _ bool, msg amqp.Publishing) error {
	r.lastExchange = exchange
	r.lastKey = key
	r.lastMsgID = msg.MessageId
	return nil
}
