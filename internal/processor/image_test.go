package processor_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/GagarinRu/avatars/internal/processor"
)

func TestProcessPNG(t *testing.T) {
	t.Parallel()
	data := makePNG(t, 200, 100)
	out, err := processor.Process(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(out.OriginalPNG) == 0 || len(out.ThumbnailPNG) == 0 {
		t.Fatal("expected encoded images")
	}
	if out.Width <= 0 || out.Height <= 0 {
		t.Fatalf("unexpected dimensions: %dx%d", out.Width, out.Height)
	}
}

func TestProcessRejectsEmpty(t *testing.T) {
	t.Parallel()
	_, err := processor.Process(bytes.NewReader(nil))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestProcessRejectsInvalid(t *testing.T) {
	t.Parallel()
	_, err := processor.Process(bytes.NewReader([]byte("not-an-image")))
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func makePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
