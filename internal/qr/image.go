package qr

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"

	// Register decoders so DecodeImageBytes handles common formats.
	_ "image/gif"
	_ "image/jpeg"
)

// isDark reports whether a pixel is "dark" using a fixed luminance threshold.
func isDark(c color.Color) bool {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return false // fully transparent = background
	}
	// Rec. 601 luma, 16-bit inputs.
	y := (299*r + 587*g + 114*b) / 1000
	return y < 0x8000
}

// DecodeImage extracts text from a QR code in an already-decoded image. It
// assumes an axis-aligned, unrotated code with a quiet zone, which matches how
// QR images are pasted into emails. Returns an error if no code is found.
func DecodeImage(img image.Image) (string, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 21 || h < 21 {
		return "", errors.New("qr: image too small")
	}
	// Binarize into a dark mask.
	dark := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if isDark(img.At(b.Min.X+x, b.Min.Y+y)) {
				dark[y*w+x] = true
			}
		}
	}
	// Bounding box of dark pixels (trims quiet zone / borders).
	minX, minY, maxX, maxY := w, h, -1, -1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if dark[y*w+x] {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if maxX < 0 {
		return "", errors.New("qr: no dark pixels")
	}
	bw := maxX - minX + 1
	bh := maxY - minY + 1

	// Estimate module size from the top-left finder pattern's top edge, which is
	// a solid run of 7 dark modules starting at the bounding-box corner.
	run := 0
	for x := minX; x <= maxX && dark[minY*w+x]; x++ {
		run++
	}
	if run < 7 {
		// Fall back: try the row a few pixels down (in case of anti-aliasing).
		yy := minY + bh/14
		run = 0
		for x := minX; x <= maxX && dark[yy*w+x]; x++ {
			run++
		}
	}
	if run < 7 {
		return "", errors.New("qr: cannot locate finder pattern")
	}
	moduleF := float64(run) / 7.0
	dim := int(float64(bw)/moduleF + 0.5)
	// Snap to a valid QR size.
	if _, ok := versionForSize(dim); !ok {
		// try nearest supported dimension using height too
		dimH := int(float64(bh)/moduleF + 0.5)
		if _, ok := versionForSize(dimH); ok {
			dim = dimH
		} else {
			return "", errors.New("qr: unsupported or undetectable dimension")
		}
	}

	m := newMatrix(dim)
	sx := float64(bw) / float64(dim)
	sy := float64(bh) / float64(dim)
	for r := 0; r < dim; r++ {
		for c := 0; c < dim; c++ {
			px := minX + int((float64(c)+0.5)*sx)
			py := minY + int((float64(r)+0.5)*sy)
			if px > maxX {
				px = maxX
			}
			if py > maxY {
				py = maxY
			}
			m.mod[m.idx(r, c)] = dark[py*w+px]
		}
	}
	return Decode(m)
}

// DecodeImageBytes decodes PNG/JPEG/GIF bytes and extracts a QR payload.
func DecodeImageBytes(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return DecodeImage(img)
}

// RenderPNG renders a matrix to a PNG at the given module scale and quiet-zone
// width (in modules). Used to synthesize test fixtures.
func RenderPNG(m *Matrix, scale, quiet int) ([]byte, error) {
	if scale < 1 {
		scale = 1
	}
	dim := (m.Size + 2*quiet) * scale
	img := image.NewGray(image.Rect(0, 0, dim, dim))
	for y := 0; y < dim; y++ {
		for x := 0; x < dim; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if !m.Get(r, c) {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					x := (c+quiet)*scale + dx
					y := (r+quiet)*scale + dy
					img.SetGray(x, y, color.Gray{Y: 0})
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
