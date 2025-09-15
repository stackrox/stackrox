package common

import "github.com/stackrox/rox/generated/storage"

type CandidateBaseImage struct {
	BaseImage *storage.BaseImage
	Layers    []*storage.BaseImageLayer
}
