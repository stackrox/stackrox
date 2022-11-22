package ternary

// Int allows you to do a ternary statement on integers.
func Int(condition bool, ifTrue, ifFalse int) int {
	if condition {
		return ifTrue
	}
	return ifFalse
}

// String allows you to do a ternary statement on strings.
func String(condition bool, ifTrue, ifFalse string) string {
	if condition {
		return ifTrue
	}
	return ifFalse
}
