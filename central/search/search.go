package search

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Indexer is the interface for search
type Indexer interface {
	AlertIndex
	ImageIndex
	PolicyIndex
	DeploymentIndex

	Close() error
}

// Result is a wrapper around the search results
type Result struct {
	ID      string
	Matches map[string][]string
	Score   float64
}

// AlertIndex provides storage functionality for alerts.
type AlertIndex interface {
	AddAlert(alert *v1.Alert) error
	DeleteAlert(id string) error
	SearchAlerts(request *v1.ParsedSearchRequest) ([]Result, error)
}

// ImageIndex provides storage functionality for images.
type ImageIndex interface {
	AddImage(image *v1.Image) error
	DeleteImage(id string) error
	SearchImages(request *v1.ParsedSearchRequest) ([]Result, error)
}

// PolicyIndex provides storage functionality for policies.
type PolicyIndex interface {
	AddPolicy(policy *v1.Policy) error
	DeletePolicy(id string) error
	SearchPolicies(request *v1.ParsedSearchRequest) ([]Result, error)
}

// DeploymentIndex provides storage functionality for deployments.
type DeploymentIndex interface {
	AddDeployment(deployment *v1.Deployment) error
	DeleteDeployment(id string) error
	SearchDeployments(request *v1.ParsedSearchRequest) ([]Result, error)
}
