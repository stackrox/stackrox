package main

import (
	"errors"
	"fmt"
	"net/http"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type imageLocation int

const (
	imageNotFound imageLocation = iota
	imageLocal
	imageRemote
)

func (l imageLocation) String() string {
	switch l {
	case imageLocal:
		return "local"
	case imageRemote:
		return "remote"
	default:
		return "not found"
	}
}

func locateImage(ref name.Reference) (imageLocation, error) {
	var fatalErr error

	if _, daemonErr := daemon.Image(ref); daemonErr != nil {
		if !cerrdefs.IsNotFound(daemonErr) {
			fatalErr = errors.Join(fatalErr, daemonErr)
		}
	} else {
		return imageLocal, nil
	}

	if _, remoteErr := remote.Head(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); remoteErr != nil {
		var transportErr *transport.Error
		if !errors.As(remoteErr, &transportErr) || transportErr.StatusCode != http.StatusNotFound {
			fatalErr = errors.Join(fatalErr, remoteErr)
		}
	} else {
		return imageRemote, nil
	}

	if fatalErr != nil {
		return imageNotFound, fmt.Errorf("locating image %s: %w", ref, fatalErr)
	}
	return imageNotFound, nil
}

func locateImageRef(registry, image, tag string) (imageLocation, error) {
	imageRef := registry + "/" + image + ":" + tag
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return imageNotFound, fmt.Errorf("parsing image reference %q: %w", imageRef, err)
	}
	return locateImage(ref)
}
