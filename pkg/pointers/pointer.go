package pointers

// Bool returns a pointer of the passed bool
//
//go:fix inline
func Bool(b bool) *bool {
	return new(b)
}

// Int32 returns a pointer of the passed int32
//
//go:fix inline
func Int32(i int32) *int32 {
	return new(i)
}

// Int64 returns a pointer of the passed int64
//
//go:fix inline
func Int64(i int64) *int64 {
	return new(i)
}

// Int returns a pointer of the passed int
//
//go:fix inline
func Int(i int) *int {
	return new(i)
}

// Float32 returns a pointer of the passed float32
//
//go:fix inline
func Float32(f float32) *float32 {
	return new(f)
}

// String returns a pointer to the passed string.
//
//go:fix inline
func String(s string) *string {
	return new(s)
}

//go:fix inline
func Pointer[T any](d T) *T {
	return new(d)
}
