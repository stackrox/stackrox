package user

// AttributeVerifier verifies that attributes meet certain conditions.
type AttributeVerifier interface {
	// Verify returns an error if provided attributes do not meet certain
	// conditions.
	Verify(attributes map[string][]string) error
}
