package permissions

import "github.com/stackrox/rox/pkg/logging"

var (
	logger = logging.LoggerForModule()
)

// A Resource represents a resource that authorization allows or disallows access to.
// Examples include clusters and integrations.
type Resource string

// Access represents the type of access to a resource.
type Access int

const (
	// ViewAccess represents view access to a resource.
	ViewAccess Access = iota
	// ModifyAccess represents modify access to a resource.
	ModifyAccess
)

// A Permission represents a level of access to a resource.
type Permission struct {
	Resource Resource
	Access   Access
}

func (a Access) String() string {
	switch a {
	case ViewAccess:
		return "view"
	case ModifyAccess:
		return "modify"
	}
	logger.Errorf("Access %d cannot be stringified, please update the String() function!", int(a))
	return ""
}

// View returns the permission struct for viewing resource r.
func View(r Resource) Permission {
	return Permission{Resource: r, Access: ViewAccess}
}

// Modify returns the permission struct for modifying resource r.
func Modify(r Resource) Permission {
	return Permission{Resource: r, Access: ModifyAccess}
}
