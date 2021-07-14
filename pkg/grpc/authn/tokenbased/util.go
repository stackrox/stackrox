package tokenbased

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/grpc/metadata"
)

// ResolvedRole with no name and global access scope.
type resolvedPseudoRoleImpl struct {
	permissions map[string]storage.Access
}

func (rpr *resolvedPseudoRoleImpl) GetRoleName() string {
	return ""
}
func (rpr *resolvedPseudoRoleImpl) GetPermissions() map[string]storage.Access {
	return rpr.permissions
}
func (rpr *resolvedPseudoRoleImpl) GetAccessScope() *storage.SimpleAccessScope {
	return nil
}

// ExtractToken extracts the token of the given type (e.g., "Bearer") from the given metadata.
func ExtractToken(md metadata.MD, tokenType string) string {
	authHeaders := md.Get("authorization")
	if len(authHeaders) != 1 {
		return ""
	}

	var tokenPrefix string
	if tokenType != "" {
		tokenPrefix = tokenType + " "
	}
	prefixLen := len(tokenPrefix)
	authHeader := authHeaders[0]
	if len(authHeader) < prefixLen || !strings.EqualFold(authHeader[:prefixLen], tokenPrefix) {
		return ""
	}

	return authHeader[prefixLen:]
}
