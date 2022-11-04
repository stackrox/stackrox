package pointers

// Bool returns a pointer of the passed bool
func Bool(b bool) *bool {
	return &b
}
