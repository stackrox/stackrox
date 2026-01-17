package certinit

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
)

var log = logging.LoggerForModule()

var (
	destinationDir  = mtls.CertsPrefix
	legacySourceDir = env.RegisterSetting("ROX_CERTS_LEGACY_DIR", env.WithDefault("/run/secrets/stackrox.io/certs-legacy/")).Setting()
	newSourceDir    = env.RegisterSetting("ROX_CERTS_NEW_DIR", env.WithDefault("/run/secrets/stackrox.io/certs-new/")).Setting()
)

const timeout = 5 * time.Minute

// Run sets up TLS certificates by copying them from new or legacy source directories
// to mtls.CertsPrefix. New certificates (from the tls-cert-sensor secret) have precedence
// over legacy certificates (from the sensor-tls secret).
// It skips initialization if these directories do not exist.
func Run() error {
	newCertsExist, err := fileutils.Exists(newSourceDir)
	if err != nil {
		log.Warnf("Error checking %q: %v", newSourceDir, err)
	}
	legacyCertsExist, err := fileutils.Exists(legacySourceDir)
	if err != nil {
		log.Warnf("Error checking %q: %v", legacySourceDir, err)
	}
	if !newCertsExist && !legacyCertsExist {
		// This should only be the case for legacy manifest-based deployments.
		log.Infof("Skipping TLS cert initialization: neither %q nor %q exist, assuming certs are mounted directly", newSourceDir, legacySourceDir)
		return nil
	}

	realDest, err := sanityCheckDestination()
	if err != nil {
		return errors.Wrapf(err, "unusable destination directory %q", destinationDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	files, err := waitForSource(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to load certificates")
	}

	if err = copyFiles(files, realDest); err != nil {
		return errors.Wrap(err, "cannot copy files")
	}

	return nil
}

func copyFiles(files []string, destDir string) error {
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return errors.Wrapf(err, "reading certificate file %s", file)
		}
		destPath := path.Join(destDir, path.Base(file))
		perm := os.FileMode(0600) // 0600 is required by Postgres (used by scanner-db)
		if err = os.WriteFile(destPath, content, perm); err != nil {
			return errors.Wrapf(err, "writing certificate file to %s", destPath)
		}
		log.Infof("Copied %q to %q", file, destPath)
	}
	return nil
}

func waitForSource(ctx context.Context) ([]string, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context expired while waiting for certificates: %w", ctx.Err())
		default:
			// Check new certificates first
			files, err := findFiles(newSourceDir)
			if err != nil {
				log.Debugf("Error checking certificates in %q: %s", newSourceDir, err)
			} else {
				log.Infof("Using %d new certificate files from %q.", len(files), newSourceDir)
				return files, nil
			}

			// Fall back to legacy certificates
			files, err = findFiles(legacySourceDir)
			if err != nil {
				log.Debugf("Error checking legacy certificates in %q: %s", legacySourceDir, err)
			} else {
				log.Infof("Using %d legacy certificates from %q.", len(files), legacySourceDir)
				return files, nil
			}

			log.Info("No certificates found. Retrying...")
			time.Sleep(5 * time.Second)
		}
	}
}

func findFiles(sourceDir string) ([]string, error) {
	realSource, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("evaluating symlinks for %q: %w", sourceDir, err)
	}

	log.Debugf("Walking %q.", realSource)
	var files []string
	err = filepath.WalkDir(realSource, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Debugf("Error accessing path %q: %s", path, walkErr)
			return nil
		}

		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") {
			if d.IsDir() {
				log.Debugf("Ignoring hidden dir %q", path)
				return filepath.SkipDir
			}

			log.Debugf("Ignoring hidden file %q", path)
			return nil
		}

		realFile, err := filepath.EvalSymlinks(path)
		if err != nil {
			log.Debugf("Ignoring file %q: %s", path, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		log.Debugf("Found file %q (%q)", path, realFile)
		files = append(files, realFile)
		return nil
	})

	if err != nil {
		return nil, errors.Wrapf(err, "walking directory %q", realSource)
	}

	minRequiredFiles := 3 // CA cert + leaf cert + private key
	if len(files) < minRequiredFiles {
		return nil, fmt.Errorf("expecting at least %d files at %q, found %d", minRequiredFiles, sourceDir, len(files))
	}

	return files, nil
}

func sanityCheckDestination() (string, error) {
	realDest, err := filepath.EvalSymlinks(destinationDir)
	if err != nil {
		return "", fmt.Errorf("evaluating symlink for %q: %w", destinationDir, err)
	}
	st, err := os.Stat(realDest)
	if err != nil {
		return "", fmt.Errorf("stat() failed for %q: %w", realDest, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%q is not a directory", realDest)
	}
	return realDest, nil
}
