package k8sintrospect

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/client-go/rest"
)

// File is a file emitted by the K8s introspection feature.
type File struct {
	Path     string
	Contents []byte
}

// FileCallback is a callback function to process a single file.
type FileCallback func(ctx concurrency.ErrorWaitable, file File) error

// Collect collects Kubernetes data relevant to the given config. If cb returns an error, processing stops and the error
// is passed through.
func Collect(ctx context.Context, collectionCfg Config, k8sClientConfig *rest.Config, cb FileCallback, since time.Time) error {
	c, err := newCollector(ctx, k8sClientConfig, collectionCfg, cb, since)
	if err != nil {
		return err
	}
	return c.Run()
}
