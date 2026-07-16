package qr

import "testing"

func TestMatrixRoundTrip(t *testing.T) {
	cases := []string{
		"http://xn--phish.example/login",
		"https://secure-update.example.test/verify?id=42",
		"HELLO WORLD 123",
		"SHORT",
	}
	for _, want := range cases {
		for mask := 0; mask < 8; mask++ {
			m, err := Encode(want, mask)
			if err != nil {
				t.Fatalf("encode %q mask %d: %v", want, mask, err)
			}
			got, err := Decode(m)
			if err != nil {
				t.Fatalf("decode %q mask %d: %v", want, mask, err)
			}
			if got != want {
				t.Fatalf("mask %d: got %q want %q", mask, got, want)
			}
		}
	}
}

func TestImageRoundTrip(t *testing.T) {
	want := "http://malware.example.test/qr-drop"
	m, err := Encode(want, 2)
	if err != nil {
		t.Fatal(err)
	}
	png, err := RenderPNG(m, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeImageBytes(png)
	if err != nil {
		t.Fatalf("decode image: %v", err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestReedSolomonCorrectsErrors(t *testing.T) {
	want := "https://recovered.example.test/x"
	m, err := Encode(want, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Flip a couple of data modules; EC level L (v?) should still recover.
	order := buildFunctionPatterns(versions[versionIndexForSize(m.Size)]).dataModuleOrder()
	m.mod[m.idx(order[0][0], order[0][1])] = !m.mod[m.idx(order[0][0], order[0][1])]
	got, err := Decode(m)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func versionIndexForSize(size int) int {
	for i, v := range versions {
		if v.size == size {
			return i
		}
	}
	return 0
}
