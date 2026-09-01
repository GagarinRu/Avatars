package objectstore_test

import (
	"testing"

	"github.com/GagarinRu/avatars/internal/objectstore"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	client, err := objectstore.NewClient(objectstore.Config{
		Endpoint:  "http://localhost:9000",
		Bucket:    "avatars",
		Region:    "us-east-1",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestReplaceURLHost(t *testing.T) {
	t.Parallel()
	got := objectstore.ReplaceURLHostForTest(
		"http://avatars.minio:9000/bucket/key?X-Amz=1",
		"http://localhost:9000",
	)
	want := "http://localhost:9000/bucket/key?X-Amz=1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestReplaceURLHostInvalid(t *testing.T) {
	t.Parallel()
	raw := "://bad"
	got := objectstore.ReplaceURLHostForTest(raw, "http://localhost:9000")
	if got != raw {
		t.Fatalf("expected unchanged invalid url")
	}
}

func TestPresignedURLEmptyKey(t *testing.T) {
	t.Parallel()
	client, err := objectstore.NewClient(objectstore.Config{
		Endpoint:  "http://localhost:9000",
		Bucket:    "avatars",
		Region:    "us-east-1",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	url, err := client.PresignedURL(t.Context(), "", 0)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if url != "" {
		t.Fatalf("expected empty url")
	}
}
