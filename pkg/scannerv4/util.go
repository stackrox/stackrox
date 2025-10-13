package scannerv4

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
)

// DigestFromImage creates a scanner v4 compatible digest from an image.
func DigestFromImage(image *storage.Image, opts ...name.Option) (name.Digest, error) {
	n := fmt.Sprintf("%s/%s@%s", image.GetName().GetRegistry(), image.GetName().GetRemote(), utils.GetSHA(image))
	digest, err := name.NewDigest(n, opts...)
	if err != nil {
		// TODO: ROX-19576: Is the assumption that images always have SHA correct?
		return name.Digest{}, fmt.Errorf("creating digest reference: %w", err)
	}

	return digest, nil
}
