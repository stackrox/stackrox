package ternary

// String allows you to do a ternary statement on strings.
func String(condition bool, ifTrue, ifFalse string) string {
	if condition {
		return ifTrue
	}
	return ifFalse
}
