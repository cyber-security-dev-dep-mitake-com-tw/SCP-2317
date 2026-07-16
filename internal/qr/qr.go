package qr

// versionInfo holds the single-block parameters we support (EC level L).
type versionInfo struct {
	version    int
	size       int // modules per side
	totalWords int
	ecWords    int // per (single) block
	alignPos   int // center of the sole alignment pattern, 0 = none (v1)
}

var versions = []versionInfo{
	{1, 21, 26, 7, 0},
	{2, 25, 44, 10, 18},
	{3, 29, 70, 15, 22},
	{4, 33, 100, 20, 26},
}

func (v versionInfo) dataWords() int { return v.totalWords - v.ecWords }

// Format information strings (15-bit, already BCH+mask), indexed by the 5-bit
// value (ecLevelBits<<3)|maskPattern. EC level L is 0b01.
var formatTable = [32]uint16{
	0x5412, 0x5125, 0x5E7C, 0x5B4B, 0x45F9, 0x40CE, 0x4F97, 0x4AA0,
	0x77C4, 0x72F3, 0x7DAA, 0x789D, 0x662F, 0x6318, 0x6C41, 0x6976,
	0x1689, 0x13BE, 0x1CE7, 0x19D0, 0x0762, 0x0255, 0x0D0C, 0x083B,
	0x355F, 0x3068, 0x3F31, 0x3A06, 0x24B4, 0x2183, 0x2EDA, 0x2BED,
}

const ecLevelL = 1

// Matrix is a square grid of modules; true = dark.
type Matrix struct {
	Size int
	mod  []bool // dark
	fn   []bool // function/reserved module (not data)
}

func newMatrix(size int) *Matrix {
	return &Matrix{Size: size, mod: make([]bool, size*size), fn: make([]bool, size*size)}
}

func (m *Matrix) idx(r, c int) int  { return r*m.Size + c }
func (m *Matrix) Get(r, c int) bool { return m.mod[m.idx(r, c)] }
func (m *Matrix) set(r, c int, dark, fn bool) {
	m.mod[m.idx(r, c)] = dark
	if fn {
		m.fn[m.idx(r, c)] = true
	}
}
func (m *Matrix) isFn(r, c int) bool { return m.fn[m.idx(r, c)] }

func (m *Matrix) placeFinder(r, c int) {
	for dr := -1; dr <= 7; dr++ {
		for dc := -1; dc <= 7; dc++ {
			rr, cc := r+dr, c+dc
			if rr < 0 || rr >= m.Size || cc < 0 || cc >= m.Size {
				continue
			}
			inRing := dr >= 0 && dr <= 6 && dc >= 0 && dc <= 6
			dark := false
			if inRing {
				if dr == 0 || dr == 6 || dc == 0 || dc == 6 {
					dark = true
				} else if dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4 {
					dark = true
				}
			}
			m.set(rr, cc, dark, true)
		}
	}
}

func (m *Matrix) placeAlignment(cr, cc int) {
	for dr := -2; dr <= 2; dr++ {
		for dc := -2; dc <= 2; dc++ {
			dark := dr == -2 || dr == 2 || dc == -2 || dc == 2 || (dr == 0 && dc == 0)
			m.set(cr+dr, cc+dc, dark, true)
		}
	}
}

// reserveFormat marks the format-info modules as function cells.
func (m *Matrix) reserveFormat() {
	for i := 0; i <= 8; i++ {
		if !m.isFn(8, i) {
			m.set(8, i, false, true)
		}
		if !m.isFn(i, 8) {
			m.set(i, 8, false, true)
		}
	}
	for i := 0; i < 8; i++ {
		m.set(8, m.Size-1-i, false, true)
		m.set(m.Size-1-i, 8, false, true)
	}
	// dark module
	m.set(m.Size-8, 8, true, true)
}

func buildFunctionPatterns(v versionInfo) *Matrix {
	m := newMatrix(v.size)
	m.placeFinder(0, 0)
	m.placeFinder(0, v.size-7)
	m.placeFinder(v.size-7, 0)
	// timing patterns
	for i := 8; i < v.size-8; i++ {
		dark := i%2 == 0
		m.set(6, i, dark, true)
		m.set(i, 6, dark, true)
	}
	if v.alignPos != 0 {
		m.placeAlignment(v.alignPos, v.alignPos)
	}
	m.reserveFormat()
	return m
}

func maskFn(pattern, r, c int) bool {
	switch pattern {
	case 0:
		return (r+c)%2 == 0
	case 1:
		return r%2 == 0
	case 2:
		return c%3 == 0
	case 3:
		return (r+c)%3 == 0
	case 4:
		return (r/2+c/3)%2 == 0
	case 5:
		return (r*c)%2+(r*c)%3 == 0
	case 6:
		return ((r*c)%2+(r*c)%3)%2 == 0
	case 7:
		return ((r+c)%2+(r*c)%3)%2 == 0
	}
	return false
}

func (m *Matrix) writeFormat(ecLevel, mask int) {
	bits := formatTable[(ecLevel<<3)|mask]
	// standard placement of the 15 format bits (bit14 = MSB)
	get := func(i int) bool { return bits&(1<<uint(14-i)) != 0 }
	// around top-left
	pos := [][2]int{{8, 0}, {8, 1}, {8, 2}, {8, 3}, {8, 4}, {8, 5}, {8, 7}, {8, 8}, {7, 8}, {5, 8}, {4, 8}, {3, 8}, {2, 8}, {1, 8}, {0, 8}}
	for i, p := range pos {
		m.mod[m.idx(p[0], p[1])] = get(i)
	}
	// duplicated copy
	for i := 0; i < 8; i++ {
		m.mod[m.idx(m.Size-1-i, 8)] = get(i)
	}
	for i := 0; i < 7; i++ {
		m.mod[m.idx(8, m.Size-7+i)] = get(8 + i)
	}
}

// dataModuleOrder returns the (r,c) traversal order for data/EC bits.
func (m *Matrix) dataModuleOrder() [][2]int {
	var order [][2]int
	up := true
	for col := m.Size - 1; col > 0; col -= 2 {
		if col == 6 {
			col = 5 // skip vertical timing column
		}
		for i := 0; i < m.Size; i++ {
			row := i
			if up {
				row = m.Size - 1 - i
			}
			for _, c := range []int{col, col - 1} {
				if !m.isFn(row, c) {
					order = append(order, [2]int{row, c})
				}
			}
		}
		up = !up
	}
	return order
}
