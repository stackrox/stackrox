package risk

import "bitbucket.org/stack-rox/apollo/generated/api/v1"

// multiplier is the interface that all risk calculations must implement
type multiplier interface {
	Score(deployment *v1.Deployment) *v1.Risk_Result
}
