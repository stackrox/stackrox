package user

// AttributeVerifier verifies that attributes meet certain conditions.
type AttributeVerifier interface {
	// Verify will verify the attributes and return an error
	// if specific checks are failing.
	Verify(attributes map[string][]string) error
}
