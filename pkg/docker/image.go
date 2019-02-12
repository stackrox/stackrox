package docker

import (
	"github.com/docker/docker/api/types/container"
)

// ImageInspect is a trimmed down version of Docker ImageInspect
type ImageInspect struct {
	ID          string   `json:"Id"`
	RepoTags    []string `json:",omitempty"`
	RepoDigests []string `json:",omitempty"`
	Config      *Config  `json:",omitempty"`
}

// Config is a trimmed down version of Docker Config
type Config struct {
	Healthcheck *container.HealthConfig `json:",omitempty"` // Healthcheck describes how to check the container is healthy
	User        string                  `json:",omitempty"`
}
