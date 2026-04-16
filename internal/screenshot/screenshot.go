package screenshot

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"

	"github.com/kbinani/screenshot"
)

const maxDimension = 1568 // Anthropic recommended max for vision

// Capture takes a screenshot of the primary display and returns it as PNG bytes.
func Capture() ([]byte, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("capture: %w", err)
	}

	resized := resizeIfNeeded(img, maxDimension)

	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

// resizeIfNeeded scales the image down so neither dimension exceeds max.
// Uses nearest-neighbour; quality is sufficient for an LLM.
func resizeIfNeeded(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}

	scale := math.Min(float64(max)/float64(w), float64(max)/float64(h))
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			sx := b.Min.X + x*w/nw
			sy := b.Min.Y + y*h/nh
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
