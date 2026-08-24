package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// makeImage creates a solid-colour source image encoded as png or jpeg.
func makeImage(t *testing.T, format string, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	var err error
	switch format {
	case "png":
		err = png.Encode(&buf, img)
	default:
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatalf("encoding test image: %v", err)
	}
	return buf.Bytes()
}

func TestResize(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		wantW, wantH int
	}{
		{"jpeg to 12x12", "jpeg", 12, 12},
		{"png to 25x25", "png", 25, 25},
		{"jpeg to 8x8", "jpeg", 8, 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := makeImage(t, tc.format, 100, 80) // 100x80 source
			out, err := Resize(src, tc.wantW, tc.wantH)
			if err != nil {
				t.Fatalf("Resize: %v", err)
			}
			img, _, err := image.Decode(bytes.NewReader(out))
			if err != nil {
				t.Fatalf("output did not decode: %v", err)
			}
			if b := img.Bounds(); b.Dx() != tc.wantW || b.Dy() != tc.wantH {
				t.Errorf("output size = %dx%d, want %dx%d", b.Dx(), b.Dy(), tc.wantW, tc.wantH)
			}
		})
	}
}

func TestResize_InvalidData(t *testing.T) {
	if _, err := Resize([]byte("not an image"), 10, 10); err == nil {
		t.Error("expected error for invalid image data, got nil")
	}
}
