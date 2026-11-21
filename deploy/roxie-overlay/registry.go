package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
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

// lookupTimeout bounds each container runtime and registry call, so an unresponsive endpoint
// fails the lookup instead of hanging it.
const lookupTimeout = 10 * time.Second

func locateImage(ctx context.Context, ref name.Reference) (imageLocation, error) {
	var fatalErr error

	if _, daemonErr := locateLocal(ctx, ref); daemonErr != nil {
		if !cerrdefs.IsNotFound(daemonErr) {
			fatalErr = errors.Join(fatalErr, daemonErr)
		}
	} else {
		return imageLocal, nil
	}

	if _, remoteErr := locateRemote(ctx, ref); remoteErr != nil {
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

func locateImageRef(ctx context.Context, registry, image, tag string) (imageLocation, error) {
	imageRef := registry + "/" + image + ":" + tag
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return imageNotFound, fmt.Errorf("parsing image reference %q: %w", imageRef, err)
	}
	return locateImage(ctx, ref)
}

func locateLocal(ctx context.Context, ref name.Reference) (v1.Image, error) {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	return daemon.Image(ref, daemon.WithContext(ctx))
}

func locateRemote(ctx context.Context, ref name.Reference) (*v1.Descriptor, error) {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	return remote.Head(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain))
}
