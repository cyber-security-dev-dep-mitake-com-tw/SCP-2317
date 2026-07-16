package qr

import (
	"errors"
	"strings"
)

// bitBuffer is a simple MSB-first bit accumulator.
type bitBuffer struct {
	bits []bool
}

func (b *bitBuffer) append(val, n int) {
	for i := n - 1; i >= 0; i-- {
		b.bits = append(b.bits, val&(1<<uint(i)) != 0)
	}
}

func (b *bitBuffer) bytes() []byte {
	out := make([]byte, (len(b.bits)+7)/8)
	for i, bit := range b.bits {
		if bit {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

func pickVersion(dataLen int) (versionInfo, bool) {
	for _, v := range versions {
		// byte mode: 4 bit mode + 8 bit count (v1-4 use 8-bit count) + 8*len
		need := 4 + 8 + 8*dataLen
		if need <= v.dataWords()*8 {
			return v, true
		}
	}
	return versionInfo{}, false
}

// Encode builds a QR matrix for the given text using byte mode, EC level L,
// and the given mask pattern. Used to synthesize test fixtures.
func Encode(text string, mask int) (*Matrix, error) {
	data := []byte(text)
	v, ok := pickVersion(len(data))
	if !ok {
		return nil, errors.New("qr: text too long for supported versions (max ~78 bytes)")
	}

	var bb bitBuffer
	bb.append(0b0100, 4)    // byte mode indicator
	bb.append(len(data), 8) // character count (8-bit for v1-4 byte mode)
	for _, c := range data {
		bb.append(int(c), 8)
	}
	capacityBits := v.dataWords() * 8
	// terminator
	term := capacityBits - len(bb.bits)
	if term > 4 {
		term = 4
	}
	if term > 0 {
		bb.append(0, term)
	}
	// pad to byte boundary
	for len(bb.bits)%8 != 0 {
		bb.bits = append(bb.bits, false)
	}
	codewords := bb.bytes()
	// pad bytes 0xEC, 0x11
	pad := []byte{0xEC, 0x11}
	for i := 0; len(codewords) < v.dataWords(); i++ {
		codewords = append(codewords, pad[i%2])
	}

	ec := rsEncode(codewords, v.ecWords)
	all := append(append([]byte(nil), codewords...), ec...)

	m := buildFunctionPatterns(v)
	order := m.dataModuleOrder()
	for i, pos := range order {
		byteIdx := i / 8
		bitIdx := 7 - i%8
		dark := false
		if byteIdx < len(all) {
			dark = all[byteIdx]&(1<<uint(bitIdx)) != 0
		}
		if maskFn(mask, pos[0], pos[1]) {
			dark = !dark
		}
		m.mod[m.idx(pos[0], pos[1])] = dark
	}
	m.writeFormat(ecLevelL, mask)
	return m, nil
}

func hamming(a, b uint16) int {
	x := a ^ b
	n := 0
	for x != 0 {
		n += int(x & 1)
		x >>= 1
	}
	return n
}

// readFormat recovers (ecLevel, mask) from the matrix format bits.
func (m *Matrix) readFormat() (int, int, bool) {
	pos := [][2]int{{8, 0}, {8, 1}, {8, 2}, {8, 3}, {8, 4}, {8, 5}, {8, 7}, {8, 8}, {7, 8}, {5, 8}, {4, 8}, {3, 8}, {2, 8}, {1, 8}, {0, 8}}
	var raw uint16
	for i, p := range pos {
		if m.Get(p[0], p[1]) {
			raw |= 1 << uint(14-i)
		}
	}
	best, bestIdx, bestDist := uint16(0), -1, 99
	for idx, f := range formatTable {
		if d := hamming(raw, f); d < bestDist {
			bestDist, bestIdx, best = d, idx, f
		}
	}
	_ = best
	if bestDist > 3 || bestIdx < 0 {
		return 0, 0, false
	}
	return bestIdx >> 3, bestIdx & 7, true
}

func versionForSize(size int) (versionInfo, bool) {
	for _, v := range versions {
		if v.size == size {
			return v, true
		}
	}
	return versionInfo{}, false
}

// Decode reads text out of a QR matrix (single block, versions 1-4, byte /
// numeric / alphanumeric segments).
func Decode(m *Matrix) (string, error) {
	v, ok := versionForSize(m.Size)
	if !ok {
		return "", errors.New("qr: unsupported matrix size")
	}
	_, mask, ok := m.readFormat()
	if !ok {
		return "", errors.New("qr: unreadable format information")
	}

	// Rebuild function map for traversal (same as encoder).
	fm := buildFunctionPatterns(v)
	order := fm.dataModuleOrder()

	bits := make([]bool, 0, len(order))
	for _, pos := range order {
		dark := m.Get(pos[0], pos[1])
		if maskFn(mask, pos[0], pos[1]) {
			dark = !dark
		}
		bits = append(bits, dark)
	}
	// pack into codewords
	words := make([]byte, len(bits)/8)
	for i := 0; i < len(words); i++ {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i*8+j] {
				b |= 1 << uint(7-j)
			}
		}
		words[i] = b
	}
	if len(words) < v.totalWords {
		return "", errors.New("qr: insufficient data modules")
	}
	block := words[:v.totalWords]
	corrected, ok := rsDecode(block, v.ecWords)
	if !ok {
		return "", errors.New("qr: reed-solomon decode failed")
	}
	return decodeSegments(corrected[:v.dataWords()])
}

const alphanumChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

func decodeSegments(data []byte) (string, error) {
	br := &bitReader{data: data}
	var sb strings.Builder
	for {
		mode, ok := br.read(4)
		if !ok || mode == 0 { // terminator or end
			break
		}
		switch mode {
		case 0b0001: // numeric
			cnt, ok := br.read(10)
			if !ok {
				return "", errors.New("qr: bad numeric length")
			}
			for cnt > 0 {
				take := 3
				if cnt < 3 {
					take = cnt
				}
				nbits := map[int]int{1: 4, 2: 7, 3: 10}[take]
				val, ok := br.read(nbits)
				if !ok {
					return "", errors.New("qr: bad numeric group")
				}
				digits := map[int]int{1: 1, 2: 2, 3: 3}[take]
				s := intToPadded(val, digits)
				sb.WriteString(s)
				cnt -= take
			}
		case 0b0010: // alphanumeric
			cnt, ok := br.read(9)
			if !ok {
				return "", errors.New("qr: bad alnum length")
			}
			for cnt >= 2 {
				val, ok := br.read(11)
				if !ok {
					return "", errors.New("qr: bad alnum pair")
				}
				sb.WriteByte(alphanumChars[val/45])
				sb.WriteByte(alphanumChars[val%45])
				cnt -= 2
			}
			if cnt == 1 {
				val, ok := br.read(6)
				if !ok {
					return "", errors.New("qr: bad alnum single")
				}
				sb.WriteByte(alphanumChars[val])
			}
		case 0b0100: // byte
			cnt, ok := br.read(8)
			if !ok {
				return "", errors.New("qr: bad byte length")
			}
			for i := 0; i < cnt; i++ {
				val, ok := br.read(8)
				if !ok {
					return "", errors.New("qr: truncated byte segment")
				}
				sb.WriteByte(byte(val))
			}
		default:
			// Unknown/unsupported mode: stop cleanly with what we have.
			return sb.String(), nil
		}
	}
	return sb.String(), nil
}

func intToPadded(val, digits int) string {
	buf := make([]byte, digits)
	for i := digits - 1; i >= 0; i-- {
		buf[i] = byte('0' + val%10)
		val /= 10
	}
	return string(buf)
}

type bitReader struct {
	data []byte
	pos  int
}

func (r *bitReader) read(n int) (int, bool) {
	if r.pos+n > len(r.data)*8 {
		return 0, false
	}
	v := 0
	for i := 0; i < n; i++ {
		byteIdx := r.pos / 8
		bitIdx := 7 - r.pos%8
		v <<= 1
		if r.data[byteIdx]&(1<<uint(bitIdx)) != 0 {
			v |= 1
		}
		r.pos++
	}
	return v, true
}
