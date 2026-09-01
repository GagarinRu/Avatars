package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	if err := os.WriteFile("testdata.png", buf.Bytes(), 0o644); err != nil {
		panic(err)
	}
}
