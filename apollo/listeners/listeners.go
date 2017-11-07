package listeners

import "bitbucket.org/stack-rox/apollo/apollo/listeners/types"

// Creator is a function stub that defined how to create a Listener
type Creator func() (types.Listener, error)

// Registry is a map of Listeners to their creation functions
var Registry = map[string]Creator{}
