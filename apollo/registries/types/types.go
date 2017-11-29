package types

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// ImageRegistry is the interface that all image registries must implement
type ImageRegistry interface {
	ProtoRegistry() *v1.Registry
	Metadata(*v1.Image) (*v1.ImageMetadata, error)
	Test() error
}
