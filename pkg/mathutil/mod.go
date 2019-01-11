package mathutil

// Mod returns the Eucledian modulus of a and b.
func Mod(a, b int) int {
	r := a % b
	if r < 0 {
		if b < 0 {
			r -= b
		} else {
			r += b
		}
	}
	return r
}
