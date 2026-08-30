package processor

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	MaxOriginalSize = 512
	ThumbSize       = 128
	MaxDimension    = 4096
)

type ProcessedImage struct {
	OriginalPNG []byte
	ThumbnailPNG []byte
	Width       int
	Height      int
	ContentType string
	SizeBytes   int64
}

func Process(reader io.Reader) (*ProcessedImage, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image")
	}
	contentType, img, err := decodeImage(data)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width > MaxDimension || height > MaxDimension {
		return nil, fmt.Errorf("image dimensions exceed maximum %d", MaxDimension)
	}
	if width < 1 || height < 1 {
		return nil, fmt.Errorf("invalid image dimensions")
	}

	original := resizeImage(img, MaxOriginalSize)
	thumb := resizeImage(img, ThumbSize)

	originalPNG, err := encodePNG(original)
	if err != nil {
		return nil, err
	}
	thumbPNG, err := encodePNG(thumb)
	if err != nil {
		return nil, err
	}

	ob := original.Bounds()
	return &ProcessedImage{
		OriginalPNG:  originalPNG,
		ThumbnailPNG: thumbPNG,
		Width:        ob.Dx(),
		Height:       ob.Dy(),
		ContentType:  contentType,
		SizeBytes:    int64(len(originalPNG)),
	}, nil
}

func decodeImage(data []byte) (string, image.Image, error) {
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		img, err := jpeg.Decode(bytes.NewReader(data))
		return "image/jpeg", img, err
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		img, err := png.Decode(bytes.NewReader(data))
		return "image/png", img, err
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", nil, fmt.Errorf("unsupported or invalid image format")
	}
	switch format {
	case "jpeg", "png", "webp":
		return "image/" + format, img, nil
	default:
		return "", nil, fmt.Errorf("unsupported image format: %s", format)
	}
}

func resizeImage(src image.Image, maxSize int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= maxSize && height <= maxSize {
		return src
	}
	var newW, newH int
	if width >= height {
		newW = maxSize
		newH = height * maxSize / width
		if newH < 1 {
			newH = 1
		}
	} else {
		newH = maxSize
		newW = width * maxSize / height
		if newW < 1 {
			newW = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}
