package clairv4

import (
	"github.com/quay/claircore"
	"github.com/stackrox/rox/generated/storage"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
)

func manifestFromImage(image *storage.Image) (*claircore.Manifest, error) {
	// TODO: ensure digests have algorithm...
	digest, err := claircore.ParseDigest(imageUtils.GetImageID(image))
	if err != nil {
		return nil, err
	}
	manifest := &claircore.Manifest{
		Hash: digest,
	}

}
