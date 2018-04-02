package risk

import "bitbucket.org/stack-rox/apollo/generated/api/v1"

// Multiplier is the interface that all risk calculations must implement
type Multiplier interface {
	Score(deployment *v1.Deployment) *v1.Risk_Result
}
