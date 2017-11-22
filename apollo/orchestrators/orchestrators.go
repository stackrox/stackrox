package orchestrators

import "bitbucket.org/stack-rox/apollo/apollo/orchestrators/types"

// Creator is a function stub that defined how to create a Orchestrator
type Creator func() (types.Orchestrator, error)

// Registry is a map of Orchestrators to their creation functions
var Registry = map[string]Creator{}
