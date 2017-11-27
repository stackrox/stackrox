package listeners

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/listeners/types"
)

// Creator is a function stub that defined how to create a Listener
type Creator func(db.DeploymentStorage) (types.Listener, error)

// Registry is a map of Listeners to their creation functions
var Registry = map[string]Creator{}
