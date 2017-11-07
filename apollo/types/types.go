package types

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// ResourceAction indicates an event type
type ResourceAction int

var (
	// DefaultRegistry defaults to dockerhub
	DefaultRegistry = "docker.io" // variable so that it could be potentially changed
)

const (
	// Create signifies create event
	Create ResourceAction = iota
	// Remove signifies Remove event
	Remove
	// Update signifies Update event
	Update
	// Unknown signifies Unknown event
	Unknown
)

// String form of ResourceAction
func (r ResourceAction) String() string {
	switch r {
	case Create:
		return "create"
	case Remove:
		return "remove"
	case Update:
		return "update"
	default:
		return "Unknown"
	}
}

// Container is a general abstraction of a docker container
type Container struct {
	ID         string
	Name       string
	Privileged bool
	Image      *v1.Image
}

// Event is the generic form of orchestrator events
type Event struct {
	Containers []*Container
	Action     ResourceAction
}

// GenerateImageFromString generates an image type from a common string format
func GenerateImageFromString(imageStr string) *v1.Image {
	var image v1.Image
	if idx := strings.Index(imageStr, "@sha256:"); idx != -1 {
		image.Sha = imageStr[idx+len("@sha256:"):]
		imageStr = imageStr[:idx]
	}
	if idx := strings.Index(imageStr, ":"); idx != -1 {
		image.Tag = imageStr[idx+1:]
		imageStr = imageStr[:idx]
	}
	if idx := strings.Index(imageStr, "/"); idx != -1 {
		image.Repo = imageStr[idx+1:]
		image.Registry = imageStr[:idx]
	} else {
		image.Repo = imageStr
		image.Registry = DefaultRegistry
	}
	return &image
}
