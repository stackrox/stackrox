package platform

import "os"

const (
	kubernetes = `kubernetes`
)

// Platform represents the platform mitigate is deployed on.
type Platform interface {
	SetupScript() string
}

// NewFromEnv returns a new platform based on configured env var.
func NewFromEnv() Platform {
	switch os.Getenv(`ROX_MITIGATE_PLATFORM`) {
	case kubernetes:
		return &kubernetesPlatform{}
	default:
		return &kubernetesPlatform{}
	}
}
