package types

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Config is the config of the registry, which can be utilized by 3rd party scanners
type Config struct {
	Username         string
	Password         string
	Insecure         bool
	URL              string
	RegistryHostname string
}

// ImageRegistry is the interface that all image registries must implement
type ImageRegistry interface {
	Match(image *v1.Image) bool
	Metadata(image *v1.Image) (*v1.ImageMetadata, error)
	Test() error
	Global() bool
	Config() *Config
}

// DockerfileInstructionSet are the set of acceptable keywords in a Dockerfile
var DockerfileInstructionSet = map[string]struct{}{
	"ADD":         {},
	"ARG":         {},
	"CMD":         {},
	"COPY":        {},
	"ENTRYPOINT":  {},
	"ENV":         {},
	"EXPOSE":      {},
	"FROM":        {},
	"HEALTHCHECK": {},
	"LABEL":       {},
	"MAINTAINER":  {},
	"ONBUILD":     {},
	"RUN":         {},
	"SHELL":       {},
	"STOPSIGNAL":  {},
	"USER":        {},
	"VOLUME":      {},
	"WORKDIR":     {},
}
