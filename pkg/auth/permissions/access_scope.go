package permissions

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// accessScopeIDPrefix should be prepended to every human-hostile ID of an
	// access scope for readability, e.g.,
	//     "io.stackrox.authz.accessscope.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	accessScopeIDPrefix = "io.stackrox.authz.accessscope."
)

// ValidateAccessScopeID returns an error if the scope ID prefix is not correct.
func ValidateAccessScopeID(scope *storage.SimpleAccessScope) error {
	if !strings.HasPrefix(scope.GetId(), accessScopeIDPrefix) {
		return errors.Errorf("id field must be in '%s*' format", accessScopeIDPrefix)
	}
	return nil
}

// GenerateAccessScopeID returns a random valid access scope ID.
func GenerateAccessScopeID() string {
	return accessScopeIDPrefix + uuid.NewV4().String()
}

// EnsureValidAccessScopeID converts id to the correct format if necessary.
func EnsureValidAccessScopeID(id string) string {
	if strings.HasPrefix(id, accessScopeIDPrefix) {
		return id
	}
	return accessScopeIDPrefix + id
}

// AccessScopeExcludeAll has empty rules and hence excludes all
// scoped resources. Global resources must be unaffected.
var AccessScopeExcludeAll = &storage.SimpleAccessScope{
	Id:          EnsureValidAccessScopeID("denyall"),
	Name:        "Deny All",
	Description: "No access to scoped resources",
	Rules:       &storage.SimpleAccessScope_Rules{},
}

// AccessScopeIncludeAll has empty rules and hence excludes all
// scoped resources. Global resources must be unaffected.
var AccessScopeIncludeAll = &storage.SimpleAccessScope{
	Id:          EnsureValidAccessScopeID("unrestricted"),
	Name:        "Unrestricted",
	Description: "Access to all clusters and namespaces",
}
