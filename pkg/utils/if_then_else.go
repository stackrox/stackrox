package utils

// IfThenElse is a ternary operator function that will return `a` if `cond` is true, otherwise it will return `b`
func IfThenElse[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
