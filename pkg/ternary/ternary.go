package ternary

// Int allows you to do a ternary statement on integers.
func Int(condition bool, ifTrue, ifFalse int) int {
	if condition {
		return ifTrue
	}
	return ifFalse
}
