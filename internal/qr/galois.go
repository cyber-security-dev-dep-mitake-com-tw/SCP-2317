// Package qr is a small, dependency-free QR Code codec used by the SCP-2317
// scanner to extract URLs hidden inside QR images (quishing detection) and,
// for tests, to synthesize fixtures without any external tooling or network.
//
// Scope: single error-correction block, versions 1-4, byte / numeric /
// alphanumeric segments. That covers the QR sizes typically pasted into
// phishing emails. It is intentionally not a general-purpose QR library.
package qr

// GF(256) arithmetic for QR Reed-Solomon (primitive polynomial 0x11d, gen 2).
var (
	gfExp [512]byte
	gfLog [256]byte
)

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = byte(x)
		gfLog[byte(x)] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11d
		}
	}
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[int(gfLog[a])+int(gfLog[b])]
}

func gfInv(a byte) byte {
	return gfExp[255-int(gfLog[a])]
}

// rsGenerator returns the generator polynomial for n EC codewords.
func rsGenerator(n int) []byte {
	g := []byte{1} // index 0 = highest-degree coefficient, kept monic
	for i := 0; i < n; i++ {
		// multiply g by (x - alpha^i)
		next := make([]byte, len(g)+1)
		for j := 0; j < len(g); j++ {
			next[j] ^= g[j]                    // shift (multiply by x)
			next[j+1] ^= gfMul(g[j], gfExp[i]) // multiply by alpha^i
		}
		g = next
	}
	return g
}

// rsEncode returns n EC codewords for the given data codewords.
func rsEncode(data []byte, n int) []byte {
	gen := rsGenerator(n)
	res := make([]byte, len(data)+n)
	copy(res, data)
	for i := 0; i < len(data); i++ {
		coef := res[i]
		if coef == 0 {
			continue
		}
		for j := 0; j < len(gen); j++ {
			res[i+j] ^= gfMul(gen[j], coef)
		}
	}
	return res[len(data):]
}

// rsDecode corrects a received codeword block (data+EC) in place, returning the
// corrected block and whether decoding succeeded. n is the number of EC bytes.
func rsDecode(recv []byte, n int) ([]byte, bool) {
	out := append([]byte(nil), recv...)

	// Syndromes S_i = recv(alpha^i).
	synd := make([]byte, n)
	hasErr := false
	for i := 0; i < n; i++ {
		var s byte
		for j := 0; j < len(out); j++ {
			s = gfMul(s, gfExp[i]) ^ out[j]
		}
		synd[i] = s
		if s != 0 {
			hasErr = true
		}
	}
	if !hasErr {
		return out, true
	}

	// Berlekamp-Massey to find the error-locator polynomial lambda (lambda[0]=1,
	// coefficients low-degree first).
	lambda := []byte{1}
	b := []byte{1}
	l := 0
	mShift := 1
	bd := byte(1)
	for i := 0; i < n; i++ {
		delta := synd[i]
		for j := 1; j <= l; j++ {
			if j < len(lambda) {
				delta ^= gfMul(lambda[j], synd[i-j])
			}
		}
		if delta == 0 {
			mShift++
			continue
		}
		coef := gfMul(delta, gfInv(bd))
		// t = lambda XOR (coef * x^mShift * b)
		t := append([]byte(nil), lambda...)
		for len(t) < len(b)+mShift {
			t = append(t, 0)
		}
		for j := 0; j < len(b); j++ {
			t[j+mShift] ^= gfMul(coef, b[j])
		}
		if 2*l <= i {
			b = append([]byte(nil), lambda...)
			bd = delta
			l = i + 1 - l
			mShift = 1
		} else {
			mShift++
		}
		lambda = t
	}

	// Chien search: error at position p (from the start of out) when
	// lambda(X_p^{-1}) == 0, with X_p = alpha^{N-1-p}.
	N := len(out)
	var errPos []int
	for p := 0; p < N; p++ {
		exp := (N - 1 - p) % 255
		xInv := gfExp[(255-exp)%255]
		var v, xp byte = 0, 1
		for _, c := range lambda {
			v ^= gfMul(c, xp)
			xp = gfMul(xp, xInv)
		}
		if v == 0 {
			errPos = append(errPos, p)
		}
	}
	if len(errPos) == 0 || len(errPos) != len(lambda)-1 {
		return out, false
	}

	// Omega = (S(x) * lambda(x)) mod x^n, S low-degree first.
	omega := make([]byte, n)
	for i := 0; i < n; i++ {
		var s byte
		for j := 0; j <= i && j < len(lambda); j++ {
			s ^= gfMul(synd[i-j], lambda[j])
		}
		omega[i] = s
	}
	for _, p := range errPos {
		exp := (N - 1 - p) % 255
		xInv := gfExp[(255-exp)%255]
		// numerator Omega(X^{-1})
		var num, xp byte = 0, 1
		for _, c := range omega {
			num ^= gfMul(c, xp)
			xp = gfMul(xp, xInv)
		}
		// denominator lambda'(X^{-1}): odd-index terms
		var den byte
		xp = byte(1)
		for k := 1; k < len(lambda); k += 2 {
			den ^= gfMul(lambda[k], xp)
			xp = gfMul(xp, gfMul(xInv, xInv))
		}
		if den == 0 {
			return out, false
		}
		// e_p = X * Omega(X^{-1}) / lambda'(X^{-1}); X = alpha^{exp}
		mag := gfMul(gfExp[exp], gfMul(num, gfInv(den)))
		out[p] ^= mag
	}

	// Verify.
	for i := 0; i < n; i++ {
		var s byte
		for j := 0; j < len(out); j++ {
			s = gfMul(s, gfExp[i]) ^ out[j]
		}
		if s != 0 {
			return out, false
		}
	}
	return out, true
}
