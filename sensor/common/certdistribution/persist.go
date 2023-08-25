package certdistribution

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// PersistCertificates persists certificates from the given bundle in the cache directory. Previously existing
// certificates that are not part of this bundle will not be overwritten.
func PersistCertificates(certBundle map[string]string) error {
	if len(certBundle) == 0 {
		return nil // nothing to do
	}
	if err := os.MkdirAll(cacheDir.Setting(), 0700); err != nil {
		return errors.Wrap(err, "could not ensure directory for certificate distribution exists")
	}

	var errs error
	for certFile, certContents := range certBundle {
		path := filepath.Join(cacheDir.Setting(), certFile)
		if err := os.WriteFile(path, []byte(certContents), 0600); err != nil {
			errs = multierror.Append(errs, errors.Wrapf(err, "failed to persist certificate file %s", certFile))
		}
	}

	return errs
}
