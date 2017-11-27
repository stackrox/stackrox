package types

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/docker/docker/reference"
)

// ResourceAction indicates an event type
type ResourceAction int

var (
	// DefaultRegistry defaults to dockerhub
	DefaultRegistry = "https://registry-1.docker.io" // variable so that it could be potentially changed
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

// DeploymentEvent is the act of creating, updating or removing a deployment.
type DeploymentEvent struct {
	Deployment *v1.Deployment
	Action     ResourceAction
}

// GenerateImageFromString generates an image type from a common string format
func GenerateImageFromString(imageStr string) *v1.Image {
	var image v1.Image

	// Check if its a sha and return if it is
	if strings.HasPrefix(imageStr, "sha256:") {
		image.Sha = strings.TrimPrefix(imageStr, "sha256:")
		return &image
	}

	// Cut off @sha256:
	if idx := strings.Index(imageStr, "@sha256:"); idx != -1 {
		image.Sha = imageStr[idx+len("@sha256:"):]
		imageStr = imageStr[:idx]
	}

	named, _ := reference.ParseNamed(imageStr)
	tag := reference.DefaultTag
	namedTagged, ok := named.(reference.NamedTagged)
	if ok {
		tag = namedTagged.Tag()
	}
	image.Remote = named.RemoteName()
	image.Tag = tag
	image.Registry = named.Hostname()
	return &image
}
