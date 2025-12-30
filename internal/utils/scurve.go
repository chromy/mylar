package utils

func rot(x int) int {
	switch x {
	case 1:
		return 2
	case 2:
		return 1
	default:
		return x
	}
}

func graycode(x int) int {
	switch x {
	case 3:
		return 2
	case 2:
		return 3
	default:
		return x
	}
}

// HilbertToPoint converts an index on the 2d Hilbert curve to a set of point coordinates.
func HilbertToPoint(curve int, h int) (int, int) {
	//  The bit widths in this function are:
	//    x, y  - curve
	//    h   - curve*2
	//    l   - 2
	//    e   - 2
	hwidth := curve * 2
	e := 0
	d := 0
	x := 0
	y := 0
	for i := 0; i < curve; i++ {
		// Extract 2 bits from h
		w := (h >> (hwidth - i*2 - 2)) & 3

		l := graycode(w)
		if d == 0 {
			l = rot(l)
		}
		l = l ^ e
		bit := 1 << (curve - i - 1)
		if l&2 != 0 {
			x |= bit
		}
		if l&1 != 0 {
			y |= bit
		}

		if w == 3 {
			e = 3 - e
		}
		if w == 0 || w == 3 {
			d ^= 1
		}
	}
	return x, y
}

// PointToHilbert converts a point on the Hilbert 2d curve to a set of coordinates.
func PointToHilbert(curve int, x int, y int) int {
	h := 0
	e := 0
	d := 0
	for i := 0; i < curve; i++ {
		// Extract 1 bit from x and y
		off := curve - i - 1
		a := (y >> off) & 1
		b := (x >> off) & 1
		l := a | (b << 1)
		l = l ^ e
		if d == 0 {
			l = rot(l)
		}
		w := graycode(l)
		if w == 3 {
			e = 3 - e
		}
		h = (h << 2) | w
		if w == 0 || w == 3 {
			d ^= 1
		}
	}
	return h
}
