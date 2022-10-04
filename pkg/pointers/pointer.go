package pointers

// Bool returns a pointer of the passed bool
func Bool(b bool) *bool {
	return &b
}

// Int32 returns a pointer of the passed int32
func Int32(i int32) *int32 {
	return &i
}

// Int64 returns a pointer of the passed int64
func Int64(i int64) *int64 {
	return &i
}

// Int returns a pointer of the passed int
func Int(i int) *int {
	return &i
}

// String returns a pointer to the passed string.
func String(s string) *string {
	return &s
}
