package worker_test

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/GagarinRu/avatars/internal/models"
	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
	"github.com/GagarinRu/avatars/internal/worker"
)

type fakeObjects struct {
	files map[string][]byte
}

func newFakeObjects() *fakeObjects {
	return &fakeObjects{files: make(map[string][]byte)}
}

func (f *fakeObjects) Upload(_ context.Context, key string, reader io.Reader, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	f.files[key] = data
	return nil
}

func (f *fakeObjects) Download(_ context.Context, key string) ([]byte, error) {
	data, ok := f.files[key]
	if !ok {
		return nil, io.EOF
	}
	return data, nil
}

func (f *fakeObjects) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.files, key)
	}
	return nil
}

func TestServiceHandleSuccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()
	objects := newFakeObjects()

	stagingKey := "staging/user-1/msg"
	objects.files[stagingKey] = makeTinyPNG(t)

	if err := store.UpsertAvatar(ctx, &models.Avatar{
		UserID:     "user-1",
		Status:     models.StatusPending,
		StagingKey: stagingKey,
		MessageID:  "msg-1",
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	svc := worker.NewService(store, objects)
	msg := queue.AvatarProcessMessage{
		MessageID:  "msg-1",
		UserID:     "user-1",
		StagingKey: stagingKey,
	}
	if err := svc.Handle(ctx, msg, amqp.Delivery{}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	avatar, err := store.GetAvatar(ctx, "user-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if avatar.Status != models.StatusReady {
		t.Fatalf("status = %s", avatar.Status)
	}
	if _, ok := objects.files["avatars/user-1/original.png"]; !ok {
		t.Fatal("original not uploaded")
	}
}

func makeTinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}
