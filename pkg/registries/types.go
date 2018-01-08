package registries

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// ImageRegistry is the interface that all image registries must implement
type ImageRegistry interface {
	ProtoRegistry() *v1.Registry
	Match(image *v1.Image) bool
	Metadata(image *v1.Image) (*v1.ImageMetadata, error)
	Test() error
	Global() bool
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
