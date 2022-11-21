package ternary

import "time"

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

// Duration allows you to do a ternary statement on time.Duration.
func Duration(condition bool, ifTrue, ifFalse time.Duration) time.Duration {
	if condition {
		return ifTrue
	}
	return ifFalse
}
