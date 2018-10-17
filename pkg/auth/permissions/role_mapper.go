package permissions

// A RoleMapper returns the role corresponding to an identifier
// obtained from a token.
type RoleMapper interface {
	Role(id string) Role
}
