package user

// AttributeVerifier verifies that attributes meet certain conditions.
// Attributes include all attributes "known" to us, they are included
// within tokens.ExternalUserClaim.
type AttributeVerifier interface {
	// Verify will verify the attributes and return an error
	// if specific checks are failing.
	Verify(attributes map[string][]string) error
}
